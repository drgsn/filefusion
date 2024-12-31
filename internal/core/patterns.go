package core

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type PatternError struct {
	Pattern string
	Reason  string
}

func (e *PatternError) Error() string {
	return fmt.Sprintf("invalid pattern %q: %s", e.Pattern, e.Reason)
}

type PatternValidator struct {
	allowBraces    bool
	allowNegation  bool
	maxPatternLen  int
	allowedSymbols []rune
	bannedPatterns []string
}

func NewPatternValidator() *PatternValidator {
	return &PatternValidator{
		allowBraces:    true,
		allowNegation:  true,
		maxPatternLen:  1000, // Reasonable default
		allowedSymbols: []rune{'*', '?', '[', ']', '{', '}', ',', '!', '/'},
		bannedPatterns: []string{
			"/**/../",  // Prevent complex traversal
			"**/.*/**", // prevent hidden files
		},
	}
}

func (v *PatternValidator) ValidatePattern(pattern string) error {
	// Check for null bytes
	if strings.Contains(pattern, "\x00") {
		return &PatternError{
			Pattern: pattern,
			Reason:  "pattern contains null bytes",
		}
	}

	// Check pattern length
	if len(pattern) > v.maxPatternLen {
		return &PatternError{
			Pattern: pattern,
			Reason:  fmt.Sprintf("pattern too long (max %d chars)", v.maxPatternLen),
		}
	}

	// Check for banned patterns
	for _, banned := range v.bannedPatterns {
		if strings.Contains(pattern, banned) {
			return &PatternError{
				Pattern: pattern,
				Reason:  fmt.Sprintf("contains banned pattern: %q", banned),
			}
		}
	}

	// Validate negation usage
	if !v.allowNegation && strings.HasPrefix(pattern, "!") {
		return &PatternError{
			Pattern: pattern,
			Reason:  "negation patterns are not allowed",
		}
	}

	// Validate brace usage
	if !v.allowBraces && (strings.Contains(pattern, "{") || strings.Contains(pattern, "}")) {
		return &PatternError{
			Pattern: pattern,
			Reason:  "brace expansion is not allowed",
		}
	}

	// Check for unbalanced braces
	if err := validateBraces(pattern); err != nil {
		return err
	}

	// Check for unbalanced brackets
	if err := validateBrackets(pattern); err != nil {
		return err
	}

	// Use doublestar's built-in validation
	if !doublestar.ValidatePattern(pattern) {
		return &PatternError{
			Pattern: pattern,
			Reason:  "invalid pattern syntax",
		}
	}

	return nil
}

func validateBraces(pattern string) error {
	stack := 0
	escaped := false

	for i, ch := range pattern {
		if escaped {
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
		case '{':
			if !escaped {
				stack++
			}
		case '}':
			if !escaped {
				stack--
				if stack < 0 {
					return &PatternError{
						Pattern: pattern,
						Reason:  fmt.Sprintf("unmatched closing brace at position %d", i),
					}
				}
			}
		}
	}

	if stack > 0 {
		return &PatternError{
			Pattern: pattern,
			Reason:  "unclosed brace",
		}
	}
	return nil
}

func validateBrackets(pattern string) error {
	stack := 0
	escaped := false

	for i, ch := range pattern {
		if escaped {
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
		case '[':
			if !escaped {
				stack++
			}
		case ']':
			if !escaped {
				stack--
				if stack < 0 {
					return &PatternError{
						Pattern: pattern,
						Reason:  fmt.Sprintf("unmatched closing bracket at position %d", i),
					}
				}
			}
		}
	}

	if stack > 0 {
		return &PatternError{
			Pattern: pattern,
			Reason:  "unclosed bracket",
		}
	}
	return nil
}

func (v *PatternValidator) splitPatterns(pattern string) []string {
	if strings.HasPrefix(pattern, "\\") {
		return []string{pattern}
	}

	var patterns []string
	var currentPattern strings.Builder
	escaped := false
	inBrace := 0

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]

		if escaped {
			currentPattern.WriteByte('\\')
			currentPattern.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
		case '{':
			if !escaped {
				inBrace++
			}
			currentPattern.WriteByte(ch)
		case '}':
			if !escaped {
				inBrace--
			}
			currentPattern.WriteByte(ch)
		case ',':
			if inBrace > 0 || escaped {
				currentPattern.WriteByte(ch)
			} else if currentPattern.Len() > 0 {
				patterns = append(patterns, currentPattern.String())
				currentPattern.Reset()
			}
		default:
			currentPattern.WriteByte(ch)
		}
	}

	if currentPattern.Len() > 0 {
		patterns = append(patterns, strings.TrimSpace(currentPattern.String()))
	}
	return patterns
}

func (v *PatternValidator) expandBracePattern(pattern string) []string {
	// Check if the pattern starts with an escaped character
	if strings.HasPrefix(pattern, "\\") {
		return []string{pattern}
	}

	escaped := false
	inBrace := 0
	start := -1

	for i := 0; i < len(pattern); i++ {
		if escaped {
			escaped = false
			continue
		}

		switch pattern[i] {
		case '\\':
			escaped = true
		case '{':
			if !escaped {
				inBrace++
				if inBrace == 1 {
					start = i
				}
			}
		case '}':
			if !escaped && inBrace > 0 {
				inBrace--
				if inBrace == 0 && start >= 0 {
					prefix := pattern[:start]
					suffix := pattern[i+1:]
					options := v.splitBraceOptions(pattern[start+1 : i])

					expanded := make([]string, len(options))
					for j, opt := range options {
						expanded[j] = prefix + opt + suffix
					}

					if len(expanded) > 0 {
						// Expand each pattern if it contains unescaped braces
						var finalExpanded []string
						for _, exp := range expanded {
							if strings.Contains(exp, "{") && !strings.Contains(exp, "\\{") {
								subExpanded := v.expandBracePattern(exp)
								finalExpanded = append(finalExpanded, subExpanded...)
							} else {
								finalExpanded = append(finalExpanded, exp)
							}
						}
						return finalExpanded
					}
				}
			}
		}
	}

	return []string{pattern}
}

func (v *PatternValidator) splitBraceOptions(content string) []string {
	var options []string
	var current strings.Builder
	escaped := false
	inBrace := 0

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if escaped {
			current.WriteByte('\\')
			current.WriteByte(ch)
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			escaped = true
		case '{':
			if !escaped {
				inBrace++
			}
			current.WriteByte(ch)
		case '}':
			if !escaped {
				inBrace--
			}
			current.WriteByte(ch)
		case ',':
			if inBrace > 0 || escaped {
				current.WriteByte(ch)
			} else {
				if current.Len() > 0 {
					options = append(options, strings.TrimSpace(current.String()))
					current.Reset()
				} else {
					options = append(options, "")
				}
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		options = append(options, strings.TrimSpace(current.String()))
	}

	return options
}

func (v *PatternValidator) ExpandPattern(pattern string) ([]string, error) {
	if pattern == "" {
		return []string{}, nil
	}

	if err := v.ValidatePattern(pattern); err != nil {
		return nil, err
	}

	// First split on top-level commas
	patterns := v.splitPatterns(pattern)
	for i, p := range patterns {
		patterns[i] = strings.TrimSpace(p)
	}
	var result []string

	// Expand each pattern
	for _, p := range patterns {
		expanded := v.expandBracePattern(p)
		result = append(result, expanded...)
	}

	return result, nil
}
