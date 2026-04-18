# Overview

Illustrious XL v1.0 is a high-resolution anime-focused generative model, built on the Stable Diffusion XL architecture. It is trained from the previous checkpoint Illustrious XL v0.1 and represents a significant leap in resolution capability — achieving a native resolution of **1536×1536**, the highest ever achieved natively within the SDXL framework at the time of its release.

The model uses the **Danbooru dataset** (up to 2023) as its core tagging syntax foundation, while also supporting natural language (NLP) prompting in hybrid form. It is provided as a **pretrained base model** with no aesthetic fine-tuning, making it an ideal foundation for LoRA training and custom fine-tuning.

## Prompting

Illustrious XL v1.0 uses **Danbooru-style tags** as its primary prompting syntax, similar to Illustrious v0.1. Version 1.0 still closely follows the Danbooru tag structure. (v1.1 onward added ~50% NLP support, and v2.0 further improved natural language understanding — but v1.0 is primarily tag-based.)

The model is **very sensitive to negative prompts**, unlike Pony models. Negative prompts significantly affect output quality and should be used actively alongside positive prompts.

### Recommended Positive Prefix
```
masterpiece, best quality, amazing quality, very aesthetic, newest,
```

Additional optional quality boosters:
```
absurdres, highres
```

### Recommended Negative Prompt
```
worst quality, bad quality, bad anatomy, bad hands, lowres, sketch, jpeg artifacts, signature, watermark, artist name, old, oldest, multiple views, censor, artistic error, artistic failure
```

### Tag Order
Follows standard Danbooru order:
```
[quality tags] [subject count: 1girl/1boy] [character name] [series] [artist] [general tags]
```

Example:
```
masterpiece, best quality, amazing quality, very aesthetic, newest, absurdres, 1girl, rem, re:zero, artist_name, smile, blue hair, maid outfit, solo, looking at viewer, upper body, white background
```

### Composition Tags
Use composition tags to control framing. Do not stack conflicting ones:
- `upper body`, `cowboy shot`, `portrait`, `full body`, `close-up`

### Safety Tags
- `general`, `sensitive`, `nsfw`, `explicit`

## Prompting Tips
- **Always use quality tags** (`masterpiece, best quality, amazing quality`) at the front — they are important for maintaining output quality.
- Use `newest` to steer toward more modern-looking art styles.
- Negative prompts are your friend. Use them actively, including Danbooru tags like `lowres`, `bad hands`, `jpeg artifacts`, `traditional media`, `watermark`, etc.
- `artistic error, artistic failure` in negative can reduce anatomical errors.
- Lower CFG = smoother/softer render. Higher CFG = more contrast and saturation.
- For dark scenes specifically, lower CFG slightly to prevent color artifacts.
- Tag dropout during training means you don't need to list every single relevant tag — focus on the most important details.

## Compatibility
- Fully compatible with LoRAs trained on Illustrious v0.1
- Supports ControlNet modules trained on v0.1
- Compatible with standard SDXL extensions (ADetailer, upscalers, etc.)
- Works in A1111/WebUI, ComfyUI, reForge, and other SDXL-compatible UIs

## Limitations
- The model has **no default style** by design — this is intentional for a base model. Outputs may look plain without quality/style tags or LoRAs.
- Knowledge cutoff is June 2024; newer characters may be poorly represented (use LoRA to address this).
- As a base model, it may produce less polished outputs than aesthetic-tuned finetuned variants (e.g., NoobAI, PonyXL-based finetuned models with aesthetic tuning).
- Not ideal for photorealism — this is an illustration/anime-focused model.

## Full Prompt Example
```
masterpiece, best quality, amazing quality, very aesthetic, newest, absurdres, 1girl, aqua (konosuba), konosuba, smile, blue hair, short hair, blue eyes, adventurer outfit, cape, solo, dynamic pose, outdoors, sky background, looking at viewer, full body
```