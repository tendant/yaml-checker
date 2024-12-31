import { createSignal } from "solid-js";

const App = () => {
  const [command, setCommand] = createSignal("");
  const [history, setHistory] = createSignal([]);
  const [feedback, setFeedback] = createSignal("");
  const [keyToCheck, setKeyToCheck] = createSignal("");
  const [checkResult, setCheckResult] = createSignal(null);

  const sendCommand = async () => {
    const response = await fetch("/api/command", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ command: command() }),
    });
    const data = await response.json();
    setFeedback(data.message);
    if (data.success) {
      setHistory([...history(), command()]);
    }
    setCommand(""); // Clear input
  };

  const checkKey = async () => {
    if (!keyToCheck()) {
      setCheckResult({ error: "Please enter a key to check" });
      return;
    }
    
    try {
      const response = await fetch(`/api/check-key?key=${encodeURIComponent(keyToCheck())}`);
      const data = await response.json();
      setCheckResult(data);
    } catch (error) {
      setCheckResult({ error: "Failed to check key" });
    }
  };

  return (
    <div>
      <h1>YAML Command Interface</h1>
      
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