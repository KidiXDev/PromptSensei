package prompting

func coreLogic() string {
	return `Core Logic:
- Tag ordering: [quality/meta/year/safety] -> [count/subject] -> [character] -> [series] -> [artist] -> [general/appearance tags] -> [outfit/accessories] -> [action/pose/expression] -> [environment/background] -> [lighting/mood] -> [camera/composition].
- Character handling: Maintain coherent identity markers (hair/eyes/traits). Do not merge unrelated characters.
- Prompt construction: Blend booru tags and natural text according to mode. Keep dense but readable comma-separated flow.`
}

func detailingInstruction() string {
	return `Enrichment & Detailing:
- Avoid "plain" or "empty" results. Sparse user inputs must be enriched with high-quality scene details.
- Background: Add specific locations, weather, or contextual objects (e.g., "in a lush forest", "cyberpunk street with neon signs").
- Lighting/Atmosphere: Use cinematic or artistic lighting (e.g., "soft sunlight", "rim lighting", "dramatic shadows", "volumetric fog").
- Composition: Add camera tools (e.g., "portrait", "wide shot", "view from side", "low angle").`
}

func retrievalGuidance() string {
	return `Retrieval Interpretation:
- confirmed_tags: High-confidence matches, usually kept.
- character_tags: Identity-critical tags, must be preserved.
- suggested_tags: Optional hints from lookup; use if they increase coherence.
- rejected_tags: Avoid unless user explicitly overrides.
- Required identity anchors from character context must be kept to ensure character consistency.`
}

func weightInstruction() string {
	return `Prompt Weighting:
- Use (tag) or (tag:weight) ONLY when mission-critical.
- Default behavior: Do NOT wrap tags in parentheses unless they are the primary focal point that needs extra emphasis. 90% of tags should NOT have weights.
- Rationale: Weights are a "emergency" tool. If overused, they lose effectiveness and create noisy prompts. Only apply to 1-3 key focal tags if the prompt exceeds 75 tokens.
- IMPORTANT: Booru tags often contain parentheses, e.g., "arona_(blue_archive)". To distinguish from weighting, you MUST escape them with backslashes if they are part of a literal tag, e.g., "arona_\(blue_archive\)". This is mandatory for tags with parentheses regardless of weighting.`
}

func outputContract() string {
	return `Output contract:
- Return only the final prompt.
- Do not include markdown.
- No explanation or reasoning text.
- Avoid duplicated tokens and self-contradictions.`
}
