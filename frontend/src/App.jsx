import { createSignal } from "solid-js";

const App = () => {
  const [command, setCommand] = createSignal("");
  const [history, setHistory] = createSignal([]);
  const [feedback, setFeedback] = createSignal("");
  const [keyToCheck, setKeyToCheck] = createSignal("");
  const [checkResult, setCheckResult] = createSignal(null);
  const [githubConfig, setGithubConfig] = createSignal({
    owner: "",
    repo: "",
    branch: "main",
    path: "",
    token: "",
  });
  const [yamlContent, setYamlContent] = createSignal("");

  const sendCommand = async () => {
    const response = await fetch("/api/command", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        command: command(),
        ...githubConfig(),
      }),
    });
    const data = await response.json();
    setFeedback(data.message);
    if (data.success) {
      setHistory([...history(), command()]);
      setYamlContent(data.content);
    }
    setCommand(""); // Clear input
  };

  const checkKey = async () => {
    if (!keyToCheck()) {
      setCheckResult({ error: "Please enter a key to check" });
      return;
    }
    
    try {
      const response = await fetch("/api/check-key", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...githubConfig(),
          key: keyToCheck(),
        }),
      });
      const data = await response.json();
      setCheckResult(data);
    } catch (error) {
      setCheckResult({ error: "Failed to check key" });
    }
  };

  return (
    <div>
      <h1>YAML GitHub Editor</h1>
      
      <div>
        <h2>GitHub Configuration</h2>
        <div>
          <input
            type="text"
            value={githubConfig().owner}
            onInput={(e) => setGithubConfig({...githubConfig(), owner: e.target.value})}
            placeholder="Repository Owner"
          />
          <input
            type="text"
            value={githubConfig().repo}
            onInput={(e) => setGithubConfig({...githubConfig(), repo: e.target.value})}
            placeholder="Repository Name"
          />
          <input
            type="text"
            value={githubConfig().branch}
            onInput={(e) => setGithubConfig({...githubConfig(), branch: e.target.value})}
            placeholder="Branch (default: main)"
          />
          <input
            type="text"
            value={githubConfig().path}
            onInput={(e) => setGithubConfig({...githubConfig(), path: e.target.value})}
            placeholder="File Path (e.g., config/app.yaml)"
          />
          <input
            type="password"
            value={githubConfig().token}
            onInput={(e) => setGithubConfig({...githubConfig(), token: e.target.value})}
            placeholder="GitHub Token"
          />
        </div>
      </div>

      <div>
        <h2>Execute Command</h2>
        <input
          type="text"
          value={command()}
          onInput={(e) => setCommand(e.target.value)}
          placeholder="Enter command (e.g., set key=value)"
        />
        <button onClick={sendCommand}>Execute</button>
        <div>
          <p>{feedback()}</p>
        </div>
      </div>

      <div>
        <h2>Check Key</h2>
        <input
          type="text"
          value={keyToCheck()}
          onInput={(e) => setKeyToCheck(e.target.value)}
          placeholder="Enter key to check"
        />
        <button onClick={checkKey}>Check</button>
        {checkResult() && (
          <div>
            {checkResult().error ? (
              <p style={{ color: "red" }}>{checkResult().error}</p>
            ) : (
              <div>
                <p>Exists: {checkResult().exists ? "Yes" : "No"}</p>
                {checkResult().exists && (
                  <p>Value Length: {checkResult().valueLength}</p>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      <div>
        <h2>Current YAML Content</h2>
        <pre style={{ background: "#f5f5f5", padding: "1rem", borderRadius: "4px" }}>
          {yamlContent()}
        </pre>
      </div>

      <div>
        <h2>Command History</h2>
        <ul>
          {history().map((cmd, index) => (
            <li key={index}>{cmd}</li>
          ))}
        </ul>
      </div>
    </div>
  );
};

export default App;