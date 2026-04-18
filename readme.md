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

## 🤖 Supported AI Providers

PromptSensei integrates with leading AI providers to power its generation engine. We support OpenAI-compatible APIs and major aggregators:

- **OpenAI**: [https://openai.com/](https://openai.com/)
- **OpenRouter**: [https://openrouter.ai/](https://openrouter.ai/)
- **Fireworks AI**: [https://fireworks.ai/](https://fireworks.ai/)
- **NanoGPT**: [https://nano-gpt.com/](https://nano-gpt.com/)

## 🚀 Getting Started

### Prerequisites
- [Go](https://go.dev/doc/install) 1.21 or higher.

## 📊 Dataset Configuration

PromptSensei requires two CSV files to power its local retrieval engine. Due to size and licensing, **these datasets are not included**. You must find them somewhere on the internet and ensure they follow the exact column format and order required for the application to parse them correctly.

The files must be placed in: `~/.config/prompt-sensei/system` (on Windows, this resolves to `C:\Users\<User>\.config\prompt-sensei\system`).

### 1. `tag.csv`
Used for tag autocompletion, alias resolution, and category identification.

- **Column Format**: `tag,category,post_count,alternative`
- **Example**: `1girl,0,7528518,"sole_female,1girls"`

### 2. `danbooru_character.csv`
Used for character core tag recognition and copyright mapping.

- **Column Format**: `character,copyright,trigger,core_tags,count,solo_count,url`
- **Example**: `hakurei_reimu,touhou,"hakurei reimu, reimu, touhou","1girl, brown_hair, long_hair, hair_bow, detached_sleeves",78109,27118,https://danbooru.donmai.us/posts?tags=hakurei_reimu`

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/kidixdev/PromptSensei.git
   cd PromptSensei
   ```

2. Build the binary:
   ```bash
   go build -o bin/prompt-sensei .
   ```

3. Run the application:
   ```bash
   ./bin/prompt-sensei
   ```

## 📅 Roadmap / Todo

We are constantly working to expand PromptSensei. Planned updates include:

### 🚀 Upcoming AI Providers
- [x] **OpenAI**
- [x] **OpenRouter**
- [x] **Fireworks AI**
- [x] **NanoGPT**
- [ ] **Anthropic (Claude)**
- [ ] **Google Gemini**
- [ ] **Mistral AI**
- [ ] **Groq**
- [ ] **Local LLMs** - Integration with Ollama and LM Studio.

### 🛠️ Planned Features
- [ ] **Image Preview**: Integration with ComfyUI for immediate prompt testing directly from the results screen.
- [ ] **Tag Relationship Map**: Visualize tag associations and character-core tag relations in a dedicated TUI explorer.
- [ ] **Weighted Tag Support**: Native UI support for applying and managing tag weights (e.g., `(tag:1.2)`).
- [ ] **Custom Style Templates**: Save and reuse complex prompt "recipes" and composition templates.
- [ ] **Batch Processing**: Load multiple ideas from a file to generate large sets of prompts in one run.
- [ ] **Theme Support**: Customizable TUI color schemes beyond the default aesthetic.

## 📄 License
This project is licensed under the MIT License - see the LICENSE file for details.
