// Package assets embeds static skill files into the binary so consul-review
// works out of the box without requiring a separate skill file installation.
package assets

import _ "embed"

// DefaultCodeReviewSkill is the bundled go-code-review skill. It is used
// automatically when code_review_skill is not set in the user's config.
//
//go:embed skills/go-code-review/SKILL.md
var DefaultCodeReviewSkill string
