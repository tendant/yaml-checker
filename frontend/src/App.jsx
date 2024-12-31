import { createSignal } from "solid-js";

const App = () => {
  const [command, setCommand] = createSignal("");
  const [history, setHistory] = createSignal([]);
  const [feedback, setFeedback] = createSignal("");

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

  return (
    <div>
      <h1>YAML Command Interface</h1>
      <input
        type="text"
        value={command()}
        onInput={(e) => setCommand(e.target.value)}
        placeholder="Enter command (e.g., set key=value)"
      />
      <button onClick={sendCommand}>Execute</button>
      <div>
        <h2>Feedback</h2>
        <p>{feedback()}</p>
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