import { createSignal, createEffect, Show, onMount } from "solid-js";

const App = () => {
  // State management
  const [activeTab, setActiveTab] = createSignal("config");
  const [command, setCommand] = createSignal("");
  const [history, setHistory] = createSignal([]);
  const [feedback, setFeedback] = createSignal({ message: "", type: "" });
  const [keyToCheck, setKeyToCheck] = createSignal("");
  const [checkResult, setCheckResult] = createSignal(null);
  const [isLoading, setIsLoading] = createSignal(false);
  const [githubConfig, setGithubConfig] = createSignal({
    owner: "",
    repo: "",
    branch: "main",
    path: "",
    token: "",
  });
  const [yamlContent, setYamlContent] = createSignal("");
  const [keyValue, setKeyValue] = createSignal({ key: "", value: "" });
  const [isConfigValid, setIsConfigValid] = createSignal(false);
  const [serverConfig, setServerConfig] = createSignal(null);
  const [availableFilePaths, setAvailableFilePaths] = createSignal([]);

  // Validate GitHub configuration
  createEffect(() => {
    const config = githubConfig();
    setIsConfigValid(
      config.owner.trim() !== "" && 
      config.repo.trim() !== "" && 
      config.branch.trim() !== "" && 
      config.path.trim() !== "" && 
      config.token.trim() !== ""
    );
  });

  // Send command to backend
  const sendCommand = async () => {
    if (!isConfigValid()) {
      setFeedback({ 
        message: "Please fill in all GitHub configuration fields", 
        type: "error" 
      });
      setActiveTab("config");
      return;
    }

    if (!command().trim()) {
      setFeedback({ 
        message: "Please enter a command", 
        type: "error" 
      });
      return;
    }

    setIsLoading(true);
    setFeedback({ message: "", type: "" });

    try {
      const response = await fetch("/api/command", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          command: command(),
          ...githubConfig(),
        }),
      });
      
      const data = await response.json();
      
      if (data.success) {
        setFeedback({ 
          message: data.message, 
          type: "success" 
        });
        setHistory([...history(), command()]);
        
        // After successful command, refresh the keys list
        fetchYamlContent();
        
        setCommand(""); // Clear input
        setKeyValue({ key: "", value: "" }); // Clear key-value inputs
      } else {
        setFeedback({ 
          message: data.message, 
          type: "error" 
        });
      }
    } catch (error) {
      setFeedback({ 
        message: `Error: ${error.message}`, 
        type: "error" 
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Check if a key exists in the YAML
  const checkKey = async () => {
    if (!isConfigValid()) {
      setFeedback({ 
        message: "Please fill in all GitHub configuration fields", 
        type: "error" 
      });
      setActiveTab("config");
      return;
    }

    if (!keyToCheck().trim()) {
      setCheckResult({ error: "Please enter a key to check" });
      return;
    }
    
    setIsLoading(true);
    setCheckResult(null);
    
    try {
      const response = await fetch(`/api/check-key?key=${encodeURIComponent(keyToCheck())}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...githubConfig(),
        }),
      });
      
      const data = await response.json();
      setCheckResult(data);
    } catch (error) {
      setCheckResult({ error: `Failed to check key: ${error.message}` });
    } finally {
      setIsLoading(false);
    }
  };

  // Set a key-value pair in the YAML
  const setKeyValuePair = () => {
    if (!keyValue().key.trim()) {
      setFeedback({ 
        message: "Please enter a key", 
        type: "error" 
      });
      return;
    }

    setCommand(`set ${keyValue().key}=${keyValue().value}`);
    sendCommand();
  };

  // Delete a key from the YAML
  const deleteKey = () => {
    if (!keyToCheck().trim()) {
      setFeedback({ 
        message: "Please enter a key to delete", 
        type: "error" 
      });
      return;
    }

    setCommand(`delete ${keyToCheck()}`);
    sendCommand();
  };

  // State for storing YAML keys
  const [yamlKeys, setYamlKeys] = createSignal([]);
  const [selectedKey, setSelectedKey] = createSignal("");

  // Fetch YAML keys without values
  const fetchYamlContent = async () => {
    if (!isConfigValid()) {
      setFeedback({ 
        message: "Please fill in all GitHub configuration fields", 
        type: "error" 
      });
      setActiveTab("config");
      return;
    }

    setIsLoading(true);
    setFeedback({ message: "", type: "" });

    try {
      // Use the dedicated content endpoint to fetch YAML keys
      const response = await fetch("/api/content", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...githubConfig(),
        }),
      });
      
      const data = await response.json();
      
      if (data.success) {
        setYamlKeys(data.keys || []);
        setFeedback({ 
          message: "YAML keys loaded successfully. You can now edit values for these keys.", 
          type: "success" 
        });
        setActiveTab("edit");
      } else {
        setFeedback({ 
          message: data.message, 
          type: "error" 
        });
      }
    } catch (error) {
      setFeedback({ 
        message: `Error: ${error.message}`, 
        type: "error" 
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Handle key selection
  const handleKeySelect = (key) => {
    setSelectedKey(key);
    setKeyValue({ key, value: "" });
  };

  // Fetch server configuration
  const fetchServerConfig = async () => {
    try {
      const response = await fetch("/api/config");
      if (response.ok) {
        const config = await response.json();
        setServerConfig(config);
        
        // Set available file paths
        if (config.filePaths && config.filePaths.length > 0) {
          setAvailableFilePaths(config.filePaths);
        }
        
        // Pre-fill GitHub configuration with server defaults
        setGithubConfig(prev => ({
          ...prev,
          owner: config.repoOwner || prev.owner,
          repo: config.repoName || prev.repo,
          branch: config.branch || prev.branch,
          // Don't set token from server config for security reasons
        }));
      }
    } catch (error) {
      console.error("Failed to fetch server configuration:", error);
    }
  };

  // Load server configuration on component mount
  onMount(() => {
    fetchServerConfig();
  });

  return (
    <div class="container">
      <h1>YAML GitHub Editor</h1>
      
      {/* Tabs */}
      <div class="tabs">
        <div 
          class={`tab ${activeTab() === "config" ? "active" : ""}`}
          onClick={() => setActiveTab("config")}
        >
          GitHub Config
        </div>
        <div 
          class={`tab ${activeTab() === "edit" ? "active" : ""}`}
          onClick={() => setActiveTab("edit")}
        >
          Edit YAML
        </div>
        <div 
          class={`tab ${activeTab() === "check" ? "active" : ""}`}
          onClick={() => setActiveTab("check")}
        >
          Check Keys
        </div>
        <div 
          class={`tab ${activeTab() === "advanced" ? "active" : ""}`}
          onClick={() => setActiveTab("advanced")}
        >
          Advanced
        </div>
        <div 
          class={`tab ${activeTab() === "history" ? "active" : ""}`}
          onClick={() => setActiveTab("history")}
        >
          History
        </div>
      </div>
      
      {/* Feedback message */}
      <Show when={feedback().message}>
        <div class={`feedback ${feedback().type}`}>
          {feedback().message}
        </div>
      </Show>

      {/* Loading indicator */}
      <Show when={isLoading()}>
        <div class="feedback">
          Processing request...
        </div>
      </Show>
      
      {/* GitHub Configuration Tab */}
      <div class={`tab-content ${activeTab() === "config" ? "active" : ""}`}>
        <div class="card">
          <h2>GitHub Configuration</h2>
          <div class="form-group">
            <label>Repository Owner</label>
            <input
              type="text"
              value={githubConfig().owner}
              onInput={(e) => setGithubConfig({...githubConfig(), owner: e.target.value})}
              placeholder="e.g., octocat"
            />
          </div>
          <div class="form-group">
            <label>Repository Name</label>
            <input
              type="text"
              value={githubConfig().repo}
              onInput={(e) => setGithubConfig({...githubConfig(), repo: e.target.value})}
              placeholder="e.g., hello-world"
            />
          </div>
          <div class="form-group">
            <label>Branch</label>
            <input
              type="text"
              value={githubConfig().branch}
              onInput={(e) => setGithubConfig({...githubConfig(), branch: e.target.value})}
              placeholder="e.g., main"
            />
          </div>
          <div class="form-group">
            <label>File Path</label>
            {availableFilePaths().length > 0 ? (
              <div>
                <select
                  value={githubConfig().path}
                  onChange={(e) => setGithubConfig({...githubConfig(), path: e.target.value})}
                >
                  <option value="">Select a file path</option>
                  {availableFilePaths().map((path) => (
                    <option value={path}>{path}</option>
                  ))}
                </select>
                <div class="form-group" style={{ marginTop: "10px" }}>
                  <label>Or enter custom path:</label>
                  <input
                    type="text"
                    value={githubConfig().path}
                    onInput={(e) => setGithubConfig({...githubConfig(), path: e.target.value})}
                    placeholder="e.g., config/app.yaml"
                  />
                </div>
              </div>
            ) : (
              <input
                type="text"
                value={githubConfig().path}
                onInput={(e) => setGithubConfig({...githubConfig(), path: e.target.value})}
                placeholder="e.g., config/app.yaml"
              />
            )}
          </div>
          <div class="form-group">
            <label>GitHub Token</label>
            <input
              type="password"
              value={githubConfig().token}
              onInput={(e) => setGithubConfig({...githubConfig(), token: e.target.value})}
              placeholder="GitHub Personal Access Token"
            />
          </div>
          <button onClick={fetchYamlContent}>Connect & Load YAML</button>
        </div>
      </div>
      
      {/* Edit YAML Tab */}
      <div class={`tab-content ${activeTab() === "edit" ? "active" : ""}`}>
        <div class="card">
          <h2>Edit YAML</h2>
          
          <div class="form-row">
            <div class="form-group" style={{ flex: "1" }}>
              <h3>Available Keys</h3>
              {yamlKeys().length === 0 ? (
                <p>No keys loaded yet. Configure GitHub settings and load a YAML file.</p>
              ) : (
                <div class="yaml-keys-list">
                  <ul class="key-tree">
                    {yamlKeys().map((key) => (
                      <li 
                        class={selectedKey() === key ? "selected" : ""}
                        onClick={() => handleKeySelect(key)}
                      >
                        {key}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
            
            <div class="form-group" style={{ flex: "1" }}>
              <h3>Set Value for Key</h3>
              <div class="key-value-editor">
                <div class="form-group">
                  <label>Selected Key</label>
                  <input
                    type="text"
                    value={keyValue().key}
                    onInput={(e) => setKeyValue({...keyValue(), key: e.target.value})}
                    placeholder="Key (e.g., app.name)"
                  />
                </div>
                <div class="form-group">
                  <label>New Value</label>
                  <input
                    type="text"
                    value={keyValue().value}
                    onInput={(e) => setKeyValue({...keyValue(), value: e.target.value})}
                    placeholder="Enter new value"
                  />
                </div>
                <button onClick={setKeyValuePair}>Update Value</button>
              </div>
              
              <div class="form-group" style={{ marginTop: "20px" }}>
                <h3>Add New Key</h3>
                <div class="key-value-editor">
                  <div class="form-group">
                    <label>New Key Path</label>
                    <input
                      type="text"
                      value={keyValue().key}
                      onInput={(e) => setKeyValue({...keyValue(), key: e.target.value})}
                      placeholder="Enter new key path (e.g., app.settings.timeout)"
                    />
                  </div>
                  <div class="form-group">
                    <label>Value</label>
                    <input
                      type="text"
                      value={keyValue().value}
                      onInput={(e) => setKeyValue({...keyValue(), value: e.target.value})}
                      placeholder="Enter value"
                    />
                  </div>
                  <button onClick={setKeyValuePair}>Add Key</button>
                </div>
              </div>
            </div>
          </div>
          
          <div class="form-group" style={{ marginTop: "20px" }}>
            <div class="note">
              <p><strong>Note:</strong> For security reasons, existing values are not displayed. You can only set new values for keys.</p>
            </div>
          </div>
        </div>
      </div>
      
      {/* Check Keys Tab */}
      <div class={`tab-content ${activeTab() === "check" ? "active" : ""}`}>
        <div class="card">
          <h2>Check & Delete Keys</h2>
          <div class="form-group">
            <label>Key to Check or Delete</label>
            <div class="key-value-editor">
              <input
                type="text"
                value={keyToCheck()}
                onInput={(e) => setKeyToCheck(e.target.value)}
                placeholder="Enter key path (e.g., app.name)"
              />
              <button onClick={checkKey}>Check</button>
              <button class="danger" onClick={deleteKey}>Delete</button>
            </div>
          </div>
          
          <Show when={checkResult()}>
            <div class="form-group">
              <h3>Check Result</h3>
              <div class="yaml-content">
                {checkResult().error ? (
                  <span style={{ color: "var(--danger-color)" }}>{checkResult().error}</span>
                ) : (
                  <div>
                    <p>
                      Key exists: 
                      <span class={`badge ${checkResult().exists ? "success" : "error"}`}>
                        {checkResult().exists ? "Yes" : "No"}
                      </span>
                    </p>
                    {checkResult().exists && (
                      <p>Value length: {checkResult().valueLength} characters</p>
                    )}
                  </div>
                )}
              </div>
            </div>
          </Show>
        </div>
      </div>
      
      {/* Advanced Tab */}
      <div class={`tab-content ${activeTab() === "advanced" ? "active" : ""}`}>
        <div class="card">
          <h2>Advanced Commands</h2>
          <p>Use YQ syntax to manipulate YAML. Examples:</p>
          <ul>
            <li><code>set app.name=MyApp</code> - Set a value</li>
            <li><code>delete app.logging</code> - Delete a key</li>
            <li><code>add newkey=value</code> - Add a new key-value pair</li>
          </ul>
          <div class="form-group">
            <label>Command</label>
            <input
              type="text"
              value={command()}
              onInput={(e) => setCommand(e.target.value)}
              placeholder="Enter command (e.g., set app.name=MyApp)"
            />
          </div>
          <button onClick={sendCommand}>Execute</button>
        </div>
      </div>
      
      {/* History Tab */}
      <div class={`tab-content ${activeTab() === "history" ? "active" : ""}`}>
        <div class="card">
          <h2>Command History</h2>
          {history().length === 0 ? (
            <p>No commands executed yet.</p>
          ) : (
            <ul class="history-list">
              {history().map((cmd, index) => (
                <li key={index}>
                  <code>{cmd}</code>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
};

export default App;
