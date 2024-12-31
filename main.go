package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
)

// CommandRequest represents the structure of an incoming command.
type CommandRequest struct {
	Command string `json:"command"` // e.g., "set key=value"
}

// CommandResponse represents the response after processing the command.
type CommandResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// KeyCheckResponse represents the response for key check endpoint
type KeyCheckResponse struct {
	Exists    bool   `json:"exists"`
	ValueLen  int    `json:"valueLength,omitempty"`
	Error     string `json:"error,omitempty"`
}

// YAML file path
const yamlFilePath = "config.yaml"

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(req.Command, " ", 2)
	if len(parts) < 2 {
		http.Error(w, "Invalid command format", http.StatusBadRequest)
		return
	}

	action, args := parts[0], parts[1]
	var cmd *exec.Cmd

	switch action {
	case "set":
		kv := strings.SplitN(args, "=", 2)
		if len(kv) != 2 {
			writeResponse(w, false, "Invalid set syntax. Use 'set key=value'")
			return
		}
		key, value := kv[0], kv[1]
		cmd = exec.Command("yq", "-i", ".\""+key+"\"=\""+value+"\"", yamlFilePath)
	case "delete":
		cmd = exec.Command("yq", "-i", "del(.\""+args+"\")", yamlFilePath)
	case "add":
		kv := strings.SplitN(args, "=", 2)
		if len(kv) != 2 {
			writeResponse(w, false, "Invalid add syntax. Use 'add key=value'")
			return
		}
		key, value := kv[0], kv[1]
		cmd = exec.Command("yq", "-i", ".\""+key+"\"=\""+value+"\"", yamlFilePath)
	default:
		writeResponse(w, false, "Unknown command. Use 'set', 'delete', or 'add'")
		return
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		writeResponse(w, false, "Command failed: "+string(output))
		return
	}

	writeResponse(w, true, "Command executed successfully")
}

func handleKeyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
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

	cmd := exec.Command("yq", "eval", ".\""+key+"\"", yamlFilePath)
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

func writeResponse(w http.ResponseWriter, success bool, message string) {
	resp := CommandResponse{
		Success: success,
		Message: message,
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