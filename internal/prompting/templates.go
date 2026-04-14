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

func outputContract() string {
	return `Output contract:
- Return only the final prompt.
- Do not include markdown.
- No explanation or reasoning text.
- Avoid duplicated tokens and self-contradictions.`
}
