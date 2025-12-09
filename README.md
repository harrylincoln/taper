# Taper

_Local network throttling daemon + Chrome extension remote_

Taper is a small macOS-friendly daemon that lets you **dial your network
quality up or down**, in real time. It exposes:

- A **local HTTP proxy** that applies throttling (latency + bandwidth
  shaping)\
- A **local REST API** (`http://127.0.0.1:5507`) for controlling
  profiles\
- A **Chrome extension** that acts as a remote control (slider,
  buttons, hotkeys)

Use it to simulate bad Wi-Fi on calls, test low-bandwidth UI behavior,
debug video streaming, or reproduce connection-sensitive bugs.

# Installation

## Option 1: Homebrew (recommended)

```bash
brew tap yourname/taper
brew install taper
```

## Option 2: Manual binary

Download binaries from Releases or build locally:

```bash
go build -o taper ./cmd/taper
```

## Option 3: Run directly from source

```bash
go run ./cmd/taper
```

---

# Getting Started

Start the daemon:

```bash
taper
```

You'll see something like:

```bash
    Proxy listening on :8807
    API listening on :5507
    ──────────────────────────────────────────────────────
     Taper daemon is running.

     To route your Mac traffic through this proxy:

     1) Find your network service name (e.g. "Wi-Fi"):
          networksetup -listallnetworkservices

     2) Enable HTTP/HTTPS proxies for that service:
          networksetup -setwebproxy "Wi-Fi" 127.0.0.1 8807
          networksetup -setsecurewebproxy "Wi-Fi" 127.0.0.1 8807

        (Replace "Wi-Fi" if your service is named differently.)

     3) To turn proxies OFF later:
          networksetup -setwebproxystate "Wi-Fi" off
          networksetup -setsecurewebproxystate "Wi-Fi" off

     API for the Chrome extension: http://127.0.0.1:5507
    ──────────────────────────────────────────────────────

This daemon does **not modify** system settings automatically.\
You explicitly enable/disable the proxy using Apple's official
`networksetup` tool.

------------------------------------------------------------------------
```

# macOS Proxy Setup (Manual)

Taper works by acting as a local proxy. To route traffic through it:

### 1. Discover your network service name

```bash
networksetup -listallnetworkservices
```

Most users will see:

    Wi-Fi

### 2. Enable HTTP + HTTPS proxies

```bash
networksetup -setwebproxy "Wi-Fi" 127.0.0.1 8807
networksetup -setsecurewebproxy "Wi-Fi" 127.0.0.1 8807
```

### 3. Disable when done

```bash
networksetup -setwebproxystate "Wi-Fi" off
networksetup -setsecurewebproxystate "Wi-Fi" off
```

---

# Chrome Extension Remote

The Chrome extension lets you control the daemon without touching a
terminal.

### What it provides:

- Slider to choose network quality level (1--10)
- Buttons: **Better** / **Worse**
- Keyboard shortcuts:
  - `Ctrl+Shift+Up` → increase level\
  - `Ctrl+Shift+Down` → decrease level
- Badge indicator reflects:
  - Current level (e.g. "7")
  - ❌ _X_ if the daemon API cannot be reached

The extension communicates with:

    http://127.0.0.1:5507/status
    http://127.0.0.1:5507/level

You must have the daemon running for the remote to work.

---

# Throttling Profiles

Profiles live in Go code (you can customize them):

Level Name Latency Download Upload

---

| Level | Name     | Latency | Download  | Upload    |
| ----- | -------- | ------- | --------- | --------- |
| 10    | Full     | 0 ms    | unlimited | unlimited |
| 8     | Good     | 80 ms   | 2 Mbps    | 1 Mbps    |
| 5     | OK       | 300 ms  | 300 kbps  | 150 kbps  |
| 3     | Bad      | 800 ms  | 64 kbps   | 32 kbps   |
| 1     | Terrible | 2000 ms | 16 kbps   | 8kbps     |

API level changes take effect immediately on all proxied connections.

---

# Testing

### Test the API

```bash
curl http://127.0.0.1:5507/status
```

### Change throttle level

```bash
curl -X POST http://127.0.0.1:5507/level \
  -H "Content-Type: application/json" \
  -d '{"level": 3}'
```

### Test the proxy (without touching macOS settings)

Baseline:

```bash
time curl https://example.com -o /dev/null
```

Through proxy:

```bash
time curl -x http://127.0.0.1:8807 https://example.com -o /dev/null
```

Switch levels and observe performance changes.

# License

MIT or similar.
