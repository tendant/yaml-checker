package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

// ServerConfig holds the server configuration from environment variables
type ServerConfig struct {
	RepoOwner   string   `json:"repoOwner"`
	RepoName    string   `json:"repoName"`
	Branch      string   `json:"branch"`
	GitHubToken string   `json:"githubToken"`
	FilePaths   []string `json:"filePaths"`
	Port        string   `json:"port"`
}

// Global server configuration
var serverConfig ServerConfig

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
	Exists   bool   `json:"exists"`
	ValueLen int    `json:"valueLength,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ContentResponse represents the response for content fetch endpoint
type ContentResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Keys    []string `json:"keys,omitempty"`
}

// initGitHubClient creates a new GitHub client with the provided token
func initGitHubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}

// enableCORS adds CORS headers to the response
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next(w, r)
	}
}

// logRequest logs information about incoming requests
func logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(startTime))
	}
}

// executeYqCommand executes a YQ command on the provided YAML content
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

		// Convert dot notation to proper YAML path notation for nested keys
		// Replace dots with proper path notation
		yamlPath := convertDotNotationToPath(key)
		cmd = exec.Command("yq", "eval", fmt.Sprintf("%s = %q", yamlPath, value), "-")
	case "delete":
		// Convert dot notation to proper YAML path notation for nested keys
		yamlPath := convertDotNotationToPath(args)
		cmd = exec.Command("yq", "eval", fmt.Sprintf("del(%s)", yamlPath), "-")
	case "add":
		kv := strings.SplitN(args, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid add syntax. Use 'add key=value'")
		}
		key, value := kv[0], kv[1]

		// Convert dot notation to proper YAML path notation for nested keys
		yamlPath := convertDotNotationToPath(key)
		cmd = exec.Command("yq", "eval", fmt.Sprintf("%s = %q", yamlPath, value), "-")
	default:
		return nil, fmt.Errorf("unknown command. Use 'set', 'delete', or 'add'")
	}

	// Set up input/output pipes
	cmd.Stdin = strings.NewReader(string(content))
	return cmd.CombinedOutput()
}

// convertDotNotationToPath converts a dot notation key to a proper YAML path
// e.g., "parent.child.key" becomes ".parent.child.key"
func convertDotNotationToPath(dotKey string) string {
	// If the key doesn't contain dots, just quote it
	if !strings.Contains(dotKey, ".") {
		return fmt.Sprintf(".%q", dotKey)
	}

	// For keys with dots, build a proper path expression
	parts := strings.Split(dotKey, ".")
	var pathBuilder strings.Builder

	for _, part := range parts {
		// Always add a dot separator
		pathBuilder.WriteString(".")
		pathBuilder.WriteString(fmt.Sprintf("%q", part))
	}

	return pathBuilder.String()
}

// getFileContent fetches a file's content from GitHub
func getFileContent(r *http.Request, req CommandRequest) (string, *github.RepositoryContent, error) {
	// Initialize GitHub client
	githubClient := initGitHubClient(req.Token)

	// Get the current file content from GitHub
	fileContent, _, _, err := githubClient.Repositories.GetContents(
		r.Context(),
		req.Owner,
		req.Repo,
		req.Path,
		&github.RepositoryContentGetOptions{Ref: req.Branch},
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch file: %v", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode content: %v", err)
	}

	return content, fileContent, nil
}

// applyServerDefaults applies server configuration defaults to the request
func applyServerDefaults(req *CommandRequest) {
	// Apply server defaults for empty fields
	if req.Owner == "" {
		req.Owner = serverConfig.RepoOwner
	}
	if req.Repo == "" {
		req.Repo = serverConfig.RepoName
	}
	if req.Branch == "" {
		req.Branch = serverConfig.Branch
	}
	if req.Token == "" {
		req.Token = serverConfig.GitHubToken
	}
}

// handleCommand processes YAML modification commands and updates GitHub
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

	// Apply server defaults
	applyServerDefaults(&req)

	// Validate request
	if req.Owner == "" || req.Repo == "" || req.Path == "" || req.Token == "" {
		writeResponse(w, false, "Missing required fields: owner, repo, path, and token are required", "")
		return
	}

	if req.Command == "" {
		writeResponse(w, false, "Command is required", "")
		return
	}

	// Get file content
	content, fileContent, err := getFileContent(r, req)
	if err != nil {
		writeResponse(w, false, err.Error(), "")
		return
	}

	// Execute the command on the content
	newContent, err := executeYqCommand([]byte(content), req.Command)
	if err != nil {
		writeResponse(w, false, fmt.Sprintf("Command failed: %v", err), "")
		return
	}

	// Initialize GitHub client
	githubClient := initGitHubClient(req.Token)

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

	// Extract keys from the updated content to return instead of the full content
	keys, err := extractYamlKeys(string(newContent))
	if err != nil {
		// If we can't extract keys, just return success without content
		writeResponse(w, true, "Command executed and changes pushed to GitHub", "")
		return
	}

	// Convert keys to a string representation for the response
	keysStr := "Updated keys:\n" + strings.Join(keys, "\n")
	writeResponse(w, true, "Command executed and changes pushed to GitHub", keysStr)
}

// handleKeyCheck checks if a key exists in a YAML file
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

	// Apply server defaults
	applyServerDefaults(&req)

	// Validate request
	if req.Owner == "" || req.Repo == "" || req.Path == "" || req.Token == "" {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  "Missing required fields: owner, repo, path, and token are required",
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

	// Get file content
	content, _, err := getFileContent(r, req)
	if err != nil {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
			Error:  err.Error(),
		})
		return
	}

	// Special case for the frontend to just check if the file exists
	if key == "__fetch_content_only__" {
		writeKeyCheckResponse(w, KeyCheckResponse{
			Exists: false,
		})
		return
	}

	// Check the key using yq with proper path handling
	yamlPath := convertDotNotationToPath(key)
	cmd := exec.Command("yq", "eval", yamlPath, "-")
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

// extractYamlKeys extracts only the keys from YAML content without values
func extractYamlKeys(content string) ([]string, error) {
	// Create a temporary file with the YAML content
	tmpFile, err := os.CreateTemp("", "yaml-keys-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Use a simpler yq command to extract all paths in the YAML document
	cmd := exec.Command("yq", "eval", "... | select(. != null) | path | join(\".\")", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to extract keys: %s: %s", err, string(output))
	}

	// Split the output by lines and filter empty lines
	keys := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// handleGetContent fetches YAML keys without values
func handleGetContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeContentResponse(w, false, "Invalid JSON payload", nil)
		return
	}

	// Apply server defaults
	applyServerDefaults(&req)

	// Validate request
	if req.Owner == "" || req.Repo == "" || req.Path == "" || req.Token == "" {
		writeContentResponse(w, false, "Missing required fields: owner, repo, path, and token are required", nil)
		return
	}

	// Get file content
	content, _, err := getFileContent(r, req)
	if err != nil {
		writeContentResponse(w, false, err.Error(), nil)
		return
	}

	// Extract only the keys from the YAML content
	keys, err := extractYamlKeys(content)
	if err != nil {
		writeContentResponse(w, false, fmt.Sprintf("Failed to extract keys: %v", err), nil)
		return
	}

	writeContentResponse(w, true, "YAML keys fetched successfully", keys)
}

// writeResponse writes a JSON response for command endpoints
func writeResponse(w http.ResponseWriter, success bool, message string, content string) {
	resp := CommandResponse{
		Success: success,
		Message: message,
		Content: content,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// writeKeyCheckResponse writes a JSON response for key check endpoint
func writeKeyCheckResponse(w http.ResponseWriter, resp KeyCheckResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// writeContentResponse writes a JSON response for content fetch endpoint
func writeContentResponse(w http.ResponseWriter, success bool, message string, keys []string) {
	resp := ContentResponse{
		Success: success,
		Message: message,
		Keys:    keys,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// loadServerConfig loads the server configuration from environment variables
func loadServerConfig() {
	// Set default values
	serverConfig = ServerConfig{
		RepoOwner:   getEnv("REPO_OWNER", ""),
		RepoName:    getEnv("REPO_NAME", ""),
		Branch:      getEnv("BRANCH", "main"),
		GitHubToken: getEnv("GITHUB_TOKEN", ""),
		Port:        getEnv("PORT", "8082"),
	}

	// Load file paths from environment variable
	filePathsStr := getEnv("FILE_PATHS", "config.yaml,config/app.yaml,deploy/values.yaml")
	serverConfig.FilePaths = strings.Split(filePathsStr, ",")

	// Log configuration (without sensitive data)
	log.Printf("Server configuration loaded:")
	log.Printf("  Repository Owner: %s", maskIfNotEmpty(serverConfig.RepoOwner))
	log.Printf("  Repository Name: %s", maskIfNotEmpty(serverConfig.RepoName))
	log.Printf("  Branch: %s", serverConfig.Branch)
	log.Printf("  GitHub Token: %s", maskIfNotEmpty(serverConfig.GitHubToken))
	log.Printf("  File Paths: %v", serverConfig.FilePaths)
	log.Printf("  Port: %s", serverConfig.Port)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// maskIfNotEmpty returns "****" if the string is not empty, otherwise returns "<not set>"
func maskIfNotEmpty(s string) string {
	if s == "" {
		return "<not set>"
	}
	return "****"
}

// getServerConfig returns the server configuration as JSON
func handleServerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Create a copy of the config without the token for security
	configCopy := serverConfig
	configCopy.GitHubToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configCopy)
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	// Load server configuration from environment variables
	loadServerConfig()

	// Set up routes with middleware
	http.HandleFunc("/api/command", logRequest(enableCORS(handleCommand)))
	http.HandleFunc("/api/check-key", logRequest(enableCORS(handleKeyCheck)))
	http.HandleFunc("/api/content", logRequest(enableCORS(handleGetContent)))
	http.HandleFunc("/api/config", logRequest(enableCORS(handleServerConfig)))

	// Start server
	port := ":" + serverConfig.Port
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
