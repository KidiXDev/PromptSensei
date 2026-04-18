# Overview

Pony Diffusion V6 XL is a versatile SDXL-based fine-tuned model developed by **PurpleSmartAI**. Despite the "Pony" name, it is a general-purpose model capable of generating anime, cartoon, furry, anthropomorphic characters, and humanoids — not limited to pony-style art. It is trained on approximately **2.6 million images** with a roughly 1:1 ratio between anime/cartoon/furry/pony datasets, and 1:1 ratio between safe/questionable/explicit content ratings. About 50% of all training images have been captioned with high-quality detailed natural language captions, giving the model strong NLP understanding alongside tag-based prompting.

All training images were manually ranked by the model author for aesthetic quality using a personal 1–5 scoring scale, forming the foundation of the unique **score-based quality system** that defines Pony Diffusion's prompting syntax. Artists' names have been removed, and an opt-in/opt-out filtering program was used for training data.

- **Developer**: PurpleSmartAI
- **Architecture**: Stable Diffusion XL fine-tune
- **Training data**: ~2.6M images (anime, cartoon, furry, pony, safe/explicit mix)
- **Specialization**: Wide range of non-photorealistic styles; especially strong at cartoon, anime, anthro, and furry aesthetics

## Generation Settings
- **Resolution**: 1024×1024 recommended. Also supports standard SDXL resolutions (832×1216, 1216×832, etc.)
- **Steps**: 25 is the official recommendation. 20–30 works well.
- **CFG Scale**: Default SDXL settings. Generally 6–7 is safe. The model is designed to work at default settings without extensive tuning.
- **Sampler**: Euler a with 25 steps is the official recommendation.
- **Clip Skip**: **Must be set to 2** (or -2 in ComfyUI notation). This is critical — without clip skip 2, outputs will be low-quality blobs.
- **Negative Prompt**: The model is **designed to not require negative prompts** in most cases. Minimal negatives or none at all is fine. Avoid long negative prompt lists.

## Prompting — The Score System

Pony Diffusion uses a **unique score-based quality system** that replaces traditional quality tags like `masterpiece` or `best quality`. These conventional quality tags have **little to no effect** on Pony — do not use them as quality boosters.

Instead, always prefix prompts with the score chain:

### Required Positive Prefix (Full)
```
score_9, score_8_up, score_7_up, score_6_up, score_5_up, score_4_up,
```

This is the recommended full prefix. A shorter version (`score_9` alone) exists but has a much weaker effect due to a training label issue — always use the full chain for best quality.

**What the scores mean:**
- `score_9`: Top ~10% of aesthetic quality in the training dataset
- `score_8_up`: Includes high-quality images
- `score_7_up`, `score_6_up`, etc.: Progressively broader quality inclusion

After the score prefix, simply describe what you want in natural language or tags.

### Source Tags
Use source tags to steer the model toward a specific visual style:
- `source_anime` — anime/2D illustration style
- `source_cartoon` — Western cartoon style
- `source_furry` — furry art style
- `source_pony` (or `anthro/feral pony`) — My Little Pony style specifically

### Rating Tags
Control content safety:
- `rating_safe` — SFW only
- `rating_questionable` — Suggestive, not explicit
- `rating_explicit` — Explicit content

To generate only safe content, include `rating_safe` in the positive prompt.

### Prompt Structure
```
score_9, score_8_up, score_7_up, score_6_up, score_5_up, score_4_up, [source tag], [rating tag], [your description or tags]
```

Example — Anime style:
```
score_9, score_8_up, score_7_up, score_6_up, source_anime, rating_safe, 1girl, long silver hair, blue eyes, winter coat, snowy forest, looking up at falling snow, soft lighting
```

Example — Cartoon style:
```
score_9, score_8_up, score_7_up, score_6_up, source_cartoon, rating_safe, 1boy, standing on top of a building, short hair, bangs, black hair, long sleeves, city skyline
```

Example — Furry style:
```
score_9, score_8_up, score_7_up, score_6_up, source_furry, rating_safe, anthro wolf, female, confident pose, leather jacket, urban background
```

## Prompting Tips
- The model understands **both natural language and tags** — you can mix them or use either exclusively.
- Do **not** use `masterpiece`, `best quality`, `hd`, or similar quality modifiers — they have no meaningful effect and may confuse the model.
- Negative prompts are largely unnecessary. Use only if you need to specifically exclude something.
- For realistic-looking anime art, add `realistic` to the prompt.
- The model sometimes generates **pseudo-signatures** in outputs — this is a known training artifact. Use inpainting or try negative prompting with `signature, text` to reduce it.
- Avoid long, complex negative prompt lists — they are counterproductive for this model.

## Style Diversity
The model supports a wide variety of non-photorealistic aesthetics including anime, cartoon (Western and Eastern), furry/anthro art, semi-realistic illustration, fantasy concept art, and more. It does not perform well for photorealism — use a photorealistic SDXL model for that.

## Compatibility
- Works in A1111/WebUI, ComfyUI, Forge, Fooocus, and other SDXL-compatible UIs
- Compatible with SDXL LoRAs (especially those trained specifically for Pony)
- Compatible with ControlNet modules designed for SDXL

## Limitations
- **No photorealism** — the model is explicitly not designed for it.
- Pseudo-signature artifacts can appear and are hard to fully remove.
- Artists' names have been scrubbed from training — direct artist style replication is not a feature.
- Standard quality tags (`masterpiece`, etc.) from other models have no meaningful effect here.
- NSFW content filtering requires explicit use of `rating_safe` in positive prompt; the model was trained on roughly equal safe/explicit data.

## Full Prompt Example
```
score_9, score_8_up, score_7_up, score_6_up, score_5_up, score_4_up, source_anime, rating_safe, 1girl, white dress, long blonde hair, green meadow, golden hour, warm sunlight, flowers, looking at viewer, peaceful expression, full body
```