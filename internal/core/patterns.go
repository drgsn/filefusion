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
			"../", // Prevent directory traversal
			"/..",
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
	for i, ch := range pattern {
		switch ch {
		case '{':
			stack++
		case '}':
			stack--
			if stack < 0 {
				return &PatternError{
					Pattern: pattern,
					Reason:  fmt.Sprintf("unmatched closing brace at position %d", i),
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
	for i, ch := range pattern {
		switch ch {
		case '[':
			stack++
		case ']':
			stack--
			if stack < 0 {
				return &PatternError{
					Pattern: pattern,
					Reason:  fmt.Sprintf("unmatched closing bracket at position %d", i),
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
	var patterns []string
	var currentPattern strings.Builder
	inBrace := 0

	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '{':
			inBrace++
			currentPattern.WriteByte(pattern[i])
		case '}':
			inBrace--
			currentPattern.WriteByte(pattern[i])
		case ',':
			if inBrace > 0 {
				currentPattern.WriteByte(pattern[i])
			} else if currentPattern.Len() > 0 {
				patterns = append(patterns, currentPattern.String())
				currentPattern.Reset()
			}
		default:
			currentPattern.WriteByte(pattern[i])
		}
	}
	if currentPattern.Len() > 0 {
		patterns = append(patterns, currentPattern.String())
	}
	return patterns
}

func (v *PatternValidator) expandBracePattern(pattern string) []string {
	idx := strings.Index(pattern, "{")
	if idx < 0 {
		return []string{pattern}
	}

	closeIdx := strings.LastIndex(pattern, "}")
	if closeIdx <= idx {
		return []string{pattern}
	}

	prefix := pattern[:idx]
	suffix := pattern[closeIdx+1:]
	options := strings.Split(pattern[idx+1:closeIdx], ",")

	result := make([]string, len(options))
	for i, opt := range options {
		result[i] = prefix + opt + suffix
	}
	return result
}

func (v *PatternValidator) ExpandPattern(pattern string) ([]string, error) {
	if pattern == "" {
		return []string{}, nil
	}

	if err := validateBraces(pattern); err != nil {
		return nil, err
	}

	if err := validateBrackets(pattern); err != nil {
		return nil, err
	}

	patterns := v.splitPatterns(pattern)
	var result []string

	for _, p := range patterns {
		expanded := v.expandBracePattern(p)
		result = append(result, expanded...)
	}

	return result, nil
}
