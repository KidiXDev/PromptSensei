# Overview

NoobAI-XL (also written as NOOBAI-XL or NAI-XL) is a text-to-image diffusion model developed by **Laxhar Dream Lab**, sponsored by BlueRun. Despite the casual name, it is not a beginner model — "Noob" is a stylistic name. It is built on the SDXL architecture using **Illustrious-xl-early-release-v0** as its base, and has been trained for a large number of epochs on the **complete Danbooru and e621 datasets** (approximately 13,000,000 images total), giving it one of the largest training corpora among anime-focused SDXL models.

NoobAI-XL excels at anime image generation, reproducing tens of thousands of anime characters and artist styles, with extensive furry/anthro knowledge from its e621 training data. It performs strongly at character-focused illustration with minimal use of LoRA.

- **Developer**: Laxhar Dream Lab (sponsored by BlueRun)
- **Base model**: Illustrious-XL early-release-v0
- **Training data**: Full Danbooru + e621 datasets (~13M images)
- **Knowledge cutoff**: ~October 23, 2024 (Danbooru); early 2024 (e621)
- **License**: fair-ai-public-license-1.0-sd + NoobAI-XL additional restrictions

## Model Variants

NoobAI-XL comes in two prediction types — this is a critical distinction:

### Epsilon-Prediction (Eps)
- Standard noise-prediction, same as most SD/SDXL models
- Produces more diverse and creative images
- Easier to set up — works in any SDXL-compatible UI without special configuration
- Versions: v0.75, v1.0, v1.1 (recommended: v1.1 or v0.75 for LoRA compatibility)

### V-Prediction (VPred)
- Generates images that follow prompts more closely
- Wider color gamut with stronger light and shadow effects; better dark scene handling
- Requires special UI configuration: needs **zero SNR** and **v-parameterization** enabled, and CFG Rescale support
- Does **not** support Karras samplers
- More powerful but harder to set up — requires a UI that explicitly supports v-pred (e.g., reForge, updated Forge, ComfyUI with correct nodes)
- Recommended CFG Rescale value: ~0.2 to prevent oversaturation or overly gray results

> If your UI does not support v-prediction settings, use the Epsilon version instead.

## Generation Settings

### Epsilon Version
- **Resolution**: Total area ~1024×1024. Recommended sizes: 768×1344, 832×1216, 896×1152, 1024×1024, 1152×896, 1216×832, 1344×768
- **Steps**: 28–40
- **CFG Scale**: 4–5 is the sweet spot. Higher than 5 causes over-contrast and oversaturation; recommended to stay at 4–5.
- **Sampler**: Euler a (recommended). Euler also works well.
- **Clip Skip**: **Not required.** Do NOT set clip skip 2 for NoobAI — leave at default (1).
- **VAE**: Baked-in — no external VAE needed.

### V-Prediction Version
- **Resolution**: Same as Epsilon
- **Steps**: ~30–40. Euler with SGM Uniform or DDPM Exponential schedulers work well; avoid Karras series.
- **CFG Scale**: 4–6 (use CFG Rescale ~0.2 to combat oversaturation)
- **Sampler**: Euler with SGM Uniform, DDIM, or DDPM Exponential. Euler a with SGM Uniform is also good. **Do not use Karras schedulers.**
- Requires zero SNR + v-parameterization settings enabled in the UI

## Prompting

NoobAI-XL uses **Danbooru-style tag prompting** as its core syntax. Unlike SDXL base, it does **not** respond strongly to natural language descriptions and works best with structured Danbooru tags. E621 tags are also supported for furry/anthro content.

- Use **commas** to separate tags: `1girl, solo, blue hair`
- **Do not use underscores** in prompts — use spaces: `blue hair` not `blue_hair`
- Escape parentheses with a backslash when needed: `ganyu \(genshin impact\)` or write as `ganyu (genshin impact), genshin impact`

### Aesthetic Tags
These are unique to NoobAI and are based on waifu-scorer aesthetic ranking:
- `very awa` — Top ~5% of images by aesthetic score. Use in positive prompt.
- `worst aesthetic` — Bottom ~5% by aesthetic score. Use in negative if needed.

### Quality Tags
Hierarchy (highest to lowest):
`masterpiece` > `best quality` > `high quality` / `good quality` > `normal quality` > `low quality` / `bad quality` > `worst quality`

Use upper tiers in positive, lower tiers in negative.

### Time Period Tags
Two formats:
- **Specific year**: `year 2024`, `year 2023`, `year 2021`, etc.
- **Period tags**:
  - `old` — 2005–2010
  - `early` — 2011–2014
  - `mid` — 2014–2017
  - `recent` — 2018–2020
  - `newest` — 2021–2024

Use `newest` in positive to steer toward modern-style art. Add `old, early` to negative to avoid dated-looking outputs.

### Safety Tags
- `general`, `sensitive`, `nsfw`, `explicit`
- Add `nsfw` to the negative prompt to filter explicit content.

### Recommended Positive Prefix
```
very awa, masterpiece, best quality, newest, highres, absurdres,
```

### Recommended Negative Prompt
```
worst quality, bad quality, low quality, lowres, normal quality, nsfw, text, signature, jpeg artifacts, bad anatomy, bad hands, old, early, copyright name, watermark, artist name, multiple views
```

For non-furry/human generations, also add:
```
mammal, anthro, furry, ambiguous form, feral, semi-anthro
```

## Tag Order
```
[1girl/1boy/1other] [character name (series), series] [artist name] [special tags] [general tags] [quality/aesthetic tags]
```
Quality tags should be placed at the end of the prompt.

### Artist Tags
Write the artist's name directly. No prefix or suffix.

### Character Tags
Use the format `character name (series), series`:
- ✅ `ganyu (genshin impact), genshin impact`
- ✅ `rem (re:zero), re:zero`

## Tips
- CFG above 5 on Epsilon quickly causes over-contrast — stay at 4–5.
- If an image looks too over-contrasted even at CFG 4–5, add `colorful` to the negative prompt.
- LoRAs trained on NoobAI v0.75 tend to perform better than those trained on v1.0 Epsilon (v1.0 can be overly contrasty).
- `very awa` is a powerful aesthetic booster — always include it in the positive.
- For dark scene generation, v-pred handles it more naturally than epsilon.
- The model understands both Danbooru and e621 tag systems; use e621 tags for furry-specific content.

## Compatibility
- Supports LoRA, ControlNet (normal, depth, canny available), and IP-Adapter
- LoRAs trained on Illustrious v0.1 generally work on NoobAI, though NoobAI-specific LoRAs are preferred
- Works in A1111/WebUI, ComfyUI, reForge, Forge (v-pred requires updated UI)
- Epsilon version works in any standard SDXL-compatible UI

## Limitations
- Natural language prompting is significantly weaker than on SDXL base or Pony — stick to Danbooru tags.
- V-pred version requires special UI configuration; not beginner-friendly without proper setup.
- E621 knowledge cutoff is only early 2024 (newer furry characters may not be recognized).
- CFG is sensitive — small increases past 5 can noticeably degrade output quality on the Epsilon version.

## Full Prompt Example
```
very awa, masterpiece, best quality, newest, highres, absurdres, 1girl, fischl (genshin impact), genshin impact, solo, long blonde hair, one eye covered, gothic dress, outdoors, moonlit night, looking at viewer, upper body
```
Negative:
```
worst quality, bad quality, low quality, lowres, nsfw, text, signature, watermark, bad anatomy, bad hands, old, early, multiple views, mammal, anthro, furry
```