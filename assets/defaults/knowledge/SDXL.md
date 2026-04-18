# Overview

Stable Diffusion XL (SDXL 1.0) is a large-scale text-to-image latent diffusion model developed by Stability AI. It represents a major architectural upgrade from SD 1.x and SD 2.x, featuring a 3x larger UNet backbone and dual text encoders (OpenCLIP ViT-bigG/14 + CLIP ViT-L). SDXL excels at photorealistic outputs, improved face generation, legible text within images, and aesthetically strong compositions. It is a general-purpose model capable of diverse styles — from photorealism to fantasy art, illustration, 3D render, and more.

SDXL consists of a **base model** and an optional **refiner model**. The base generates the initial latent, and the refiner adds high-frequency detail through a noising-denoising pass. The refiner is optional but can improve fine detail in some outputs.

## Architecture Highlights
- Dual text encoders: understands both natural language descriptions and keyword/tag-based prompts
- Native 1024×1024 resolution
- Two-stage pipeline: base + refiner (refiner is optional)
- Supports image-to-image, inpainting, and outpainting

## Generation Settings
- **Resolution**: Native 1024×1024. Also works at 1152×896, 896×1152, 1216×832, 832×1216. Do not go below 512×512.
- **CFG Scale**: 6–8 is the sweet spot. Too low = washed out. Too high = oversaturated artifacts.
- **Steps**: 20–30 is the quality plateau. 25 is a solid default. Going beyond 50 rarely helps.
- **Sampler**: DPM++ 2M Karras (fast and reliable), Euler a (softer, good for artistic styles)
- **Refiner**: Optional. Use on base outputs for extra fine detail, especially for photorealism. Costs extra VRAM and time.
- **VRAM**: Requires at least 8GB VRAM. 12GB+ recommended for comfortable operation.
- **Clip Skip**: Leave at default (1) for SDXL base. Do NOT set clip skip 2 — that is for Pony/Illustrious models.

## Prompting

SDXL uses dual text encoders and understands **natural language significantly better than SD 1.x**. It responds well to descriptive sentences, not just keyword lists.

### Key Differences from SD 1.5
- Write in natural language, not "tag soup." Full sentences or descriptive phrases work better.
- Quality boost tags like `masterpiece, best quality, highly detailed` have **less effect** than on SD 1.5 and can sometimes be counterproductive — SDXL's training already assumes high quality.
- Negative prompts are less critical. Keep them short and specific (e.g., `blurry, cartoon` if you want photorealism).
- Keyword weighting still works: `(keyword:1.2)` increases emphasis by 20%. Keep weights between 0.8–1.4; going higher is rarely useful.

### Prompting Style
- Think like a photographer, filmmaker, or art director.
- Describe subject, lighting, style, mood, and composition.
- Example: *"Cinematic portrait of a woman with auburn hair, soft studio lighting, shot on 85mm lens, film grain, shallow depth of field, editorial photography style"*
- For anime/illustration styles, use SDXL-based finetuned checkpoints (e.g., AnimagineXL, Pony, Illustrious), not vanilla SDXL base.

### Negative Prompt
- SDXL handles this with minimal keywords. Common negatives:
  - `blurry, low quality, watermark, signature, text` — general cleanup
  - `cartoon` — when aiming for realism
  - `extra fingers, bad anatomy` — if anatomy is an issue

## Prompt Structure
SDXL has no strict mandatory tag structure unlike Danbooru-based models. Suggested loose structure:

`[Subject description] [environment/background] [lighting] [style/medium] [quality/camera notes]`

Example:
```
A fantasy knight in ornate silver armor standing at the edge of a cliff, stormy sky behind them, dramatic rim lighting, digital painting, highly detailed, 4k
```

Or natural language style:
```
A cinematic shot of a young woman in a red dress walking through a rainy Tokyo street at night, neon reflections on wet pavement, film photography style
```

## Limitations
- Less effective with anime-specific Danbooru tags compared to anime-focused finetuned models.
- Quality boost tags (`masterpiece`, `best quality`) have weak effect on vanilla SDXL.
- The base SDXL has a somewhat "neutral" default style — finetuned checkpoints are usually preferred for specific aesthetics.
- Text rendering is improved over SD 1.5 but still unreliable for long strings.
- Pseudo-signatures can appear in some outputs (inherited from training data).

## Tips
- For a specific aesthetic style, use a finetuned SDXL checkpoint or LoRA rather than prompting the base model harder.
- The refiner is worth trying for photorealistic portrait work but skip it for fast iteration.
- Keyword weighting is supported: `(red dress:1.3)` — but don't exceed 1.5 to avoid artifacts.