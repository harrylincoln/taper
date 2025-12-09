const API_BASE = "http://127.0.0.1:5507";

async function getStatus() {
  const res = await fetch(`${API_BASE}/status`);
  if (!res.ok) throw new Error("Status error");
  return res.json();
}

async function setLevel(level) {
  await fetch(`${API_BASE}/level`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ level })
  });
}

function clampLevel(level) {
  if (level < 1) return 1;
  if (level > 10) return 10;
  return level;
}

async function changeLevel(delta) {
  try {
    const status = await getStatus();
    const current = status.level || 10;
    const next = clampLevel(current + delta);
    await setLevel(next);
    chrome.action.setBadgeText({ text: String(next) });
  } catch (err) {
    console.error("Failed to change level:", err);
    // Show an "X" badge if we can't talk to the daemon
    chrome.action.setBadgeText({ text: "X" });
  }
}

chrome.commands.onCommand.addListener((command) => {
  if (command === "throttle_up") {
    changeLevel(+1);
  } else if (command === "throttle_down") {
    changeLevel(-1);
  }
});
