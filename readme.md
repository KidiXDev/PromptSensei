# ⛩️ PromptSensei

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Version](https://img.shields.io/badge/Version-v1.3.1-81?style=flat-square)](internal/domain/version.go)

**PromptSensei** is a high-performance terminal tool designed to craft and enhance AI image generation prompts. It combines a powerful local **Booru-aware retrieval engine** with modern Large Language Models to transform simple ideas into detailed, high-quality prompts tailored for models like Stable Diffusion, Midjourney, and Pony Diffusion.

---

## ✨ Key Features

- **🧠 Local Retrieval Engine**: Blazing-fast SQLite-backed search over massive Booru datasets. Automatically resolves character core tags, tag aliases, and suggested associations without needing an internet connection.
- **🎭 Multi-Mode Prompting**:
  - **Natural**: Descriptive, weighted English sentences for high-end LLMs.
  - **Booru**: Strictly comma-separated tag strings optimized for traditional image models.
  - **Hybrid**: A balanced blend of natural language and structured tags.
- **🗂️ Knowledge Base Integration**: Inject custom Markdown documentation (characters, styles, lore) directly into the prompting context using `ctrl+k` in the editor.
- **⚙️ Modern TUI**: A beautiful, edge-to-edge terminal interface powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).


## 🚀 Getting Started

### Prerequisites
- [Go](https://go.dev/doc/install) 1.21 or higher.
- A Booru-compatible CSV dataset (placed in the `~/.config/prompt-sensei/system` directory).

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/kidixdev/PromptSensei.git
   cd PromptSensei
   ```

2. Build the binary:
   ```bash
   go build -o bin/prompt-sensei ./cmd/prompt-sensei
   ```

3. Run the application:
   ```bash
   ./bin/prompt-sensei
   ```


## 📄 License
This project is licensed under the MIT License - see the LICENSE file for details.

*Crafted with ❤️ for the AI Art Community.*
