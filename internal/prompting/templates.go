package prompting

func outputContract() string {
	return `Output contract:
- Return only the final prompt.
- Do not include markdown.
- No explanation or reasoning text.
- Avoid duplicated tokens and self-contradictions.`
}
