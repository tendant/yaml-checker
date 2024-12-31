package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// CommandRequest represents the structure of an incoming command.
type CommandRequest struct {
	Command string `json:"command"` // e.g., "set key=value"
	Owner   string `json:"owner"`   // GitHub owner
	Repo    string `json:"repo"`    // GitHub repo
	Branch  string `json:"branch"`  // GitHub branch
	Path    string `json:"path"`    // File path in repo
	Token   string `json:"token"`   // GitHub token
}

// CommandResponse represents the response after processing the command.
type CommandResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Content string `json:"content,omitempty"`
}

// KeyCheckResponse represents the response for key check endpoint
type KeyCheckResponse struct {
	Exists    bool   `json:"exists"`
	ValueLen  int    `json:"valueLength,omitempty"`
	Error     string `json:"error,omitempty"`
}

var githubClient *github.Client

func initGitHubClient(token string) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	githubClient = github.NewClient(tc)
}

func executeYqCommand(content []byte, command string) ([]byte, error) {
	// Create a temporary command that operates on the content directly
	parts := strings.SplitN(command, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid command format")
	}

	action, args := parts[0], parts[1]
	var cmd *exec.Cmd

	switch action {
	case "set":
		kv := strings.SplitN(args, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid set syntax. Use 'set key=value'")
		}
		key, value := kv[0], kv[1]
		cmd = exec.Command("yq", "eval", fmt.Sprintf(".%q = %q", key, value), "-")
	case "delete":
		cmd = exec.Command("yq", "eval", fmt.Sprintf("del(.%q)", args), "-")
	case "add":
		kv := strings.SplitN(args, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid add syntax. Use 'add key=value'")
		}
		key, value := kv[0], kv[1]
		cmd = exec.Command("yq", "eval", fmt.Sprintf(".%q = %q", key, value), "-")
	default:
		return nil, fmt.Errorf("unknown command. Use 'set', 'delete', or 'add'")
	}

	// Set up input/output pipes
	cmd.Stdin = strings.NewReader(string(content))
	return cmd.CombinedOutput()
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, false, "Invalid JSON payload", "")
		return
	}

	// Initialize GitHub client
	initGitHubClient(req.Token)

	// Get the current file content from GitHub
	fileContent, _, _, err := githubClient.Repositories.GetContents(
		r.Context(),
		req.Owner,
		req.Repo,
		req.Path,
		&github.RepositoryContentGetOptions{Ref: req.Branch},
	)
	if err != nil {
		writeResponse(w, false, fmt.Sprintf("Failed to fetch file: %v", err), "")
		return
	}

	content, err := fileContent.GetContent()
	if err != nil {
		writeResponse(w, false, fmt.Sprintf("Failed to decode content: %v", err), "")
		return
	}

	// Execute the command on the content
	newContent, err := executeYqCommand([]byte(content), req.Command)
	if err != nil {
		writeResponse(w, false, fmt.Sprintf("Command failed: %v", err), "")
		return
	}

	// Update the file in GitHub
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("Update YAML via yaml-checker: %s", req.Command)),
		Content: newContent,
		SHA:     fileContent.SHA,
		Branch:  github.String(req.Branch),
	}

	_, _, err = githubClient.Repositories.UpdateFile(
		r.Context(),
		req.Owner,
		req.Repo,
		req.Path,
		opts,
	)
	if err != nil {
		writeResponse(w, false, fmt.Sprintf("Failed to update file: %v", err), "")
		return
	}

	writeResponse(w, true, "Command executed and changes pushed to GitHub", string(newContent))
}

func handleKeyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  "Invalid request body",
		})
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  "Key parameter is required",
		})
		return
	}

	// Initialize GitHub client
	initGitHubClient(req.Token)

	// Get the file content from GitHub
	fileContent, _, _, err := githubClient.Repositories.GetContents(
		r.Context(),
		req.Owner,
		req.Repo,
		req.Path,
		&github.RepositoryContentGetOptions{Ref: req.Branch},
	)
	if err != nil {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  fmt.Sprintf("Failed to fetch file: %v", err),
		})
		return
	}

	content, err := fileContent.GetContent()
	if err != nil {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  fmt.Sprintf("Failed to decode content: %v", err),
		})
		return
	}

	// Check the key using yq
	cmd := exec.Command("yq", "eval", fmt.Sprintf(".%q", key), "-")
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  "Error checking key: " + string(output),
		})
		return
	}

	value := strings.TrimSpace(string(output))
	if value == "null" {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
		})
		return
	}

	writeKeyCheckResponse(w, KeyCheckResponse{
		Exists:   true,
		ValueLen: len(value),
	})
}

func writeResponse(w http.ResponseWriter, success bool, message string, content string) {
	resp := CommandResponse{
		Success: success,
		Message: message,
		Content: content,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeKeyCheckResponse(w http.ResponseWriter, resp KeyCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/api/command", handleCommand)
	http.HandleFunc("/api/check-key", handleKeyCheck)
	http.ListenAndServe(":8080", nil)
}