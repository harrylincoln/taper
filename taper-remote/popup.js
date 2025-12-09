const API_BASE = "http://127.0.0.1:5507";

const levelSpan = document.getElementById("level");
const slider = document.getElementById("slider");
const upBtn = document.getElementById("up");
const downBtn = document.getElementById("down");

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

async function refreshUI() {
  try {
    const status = await getStatus();
    const level = status.level || 10;
    levelSpan.textContent = level;
    slider.value = level;
    setControlsEnabled(true);
  } catch (err) {
    console.error("Failed to fetch status:", err);
    levelSpan.textContent = "No daemon";
    setControlsEnabled(false);
  }
}

slider.addEventListener("input", async (e) => {
  const level = clampLevel(parseInt(e.target.value, 10));
  levelSpan.textContent = level;
  try {
    await setLevel(level);
  } catch (err) {
    console.error("Failed to set level:", err);
  }
});

upBtn.addEventListener("click", async () => {
  const current = parseInt(slider.value, 10) || 10;
  const next = clampLevel(current + 1);
  slider.value = next;
  levelSpan.textContent = next;
  await setLevel(next);
});

downBtn.addEventListener("click", async () => {
  const current = parseInt(slider.value, 10) || 10;
  const next = clampLevel(current - 1);
  slider.value = next;
  levelSpan.textContent = next;
  await setLevel(next);
});

// Load current level when popup opens
refreshUI();
