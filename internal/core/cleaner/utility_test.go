package cleaner

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	options := DefaultOptions()

	if options == nil {
		t.Error("DefaultOptions() returned nil")
	}

	// Test default values
	if !options.RemoveComments {
		t.Error("RemoveComments should be true by default")
	}
	if !options.PreserveDocComments {
		t.Error("PreserveDocComments should be true by default")
	}
	if options.RemoveImports {
		t.Error("RemoveImports should be false by default")
	}
	if !options.RemoveLogging {
		t.Error("RemoveLogging should be true by default")
	}
	if !options.RemoveGettersSetters {
		t.Error("RemoveGettersSetters should be true by default")
	}
	if !options.OptimizeWhitespace {
		t.Error("OptimizeWhitespace should be true by default")
	}
	if !options.RemoveEmptyLines {
		t.Error("RemoveEmptyLines should be true by default")
	}

	// Test logging prefixes
	if options.LoggingPrefixes == nil {
		t.Error("LoggingPrefixes should not be nil")
	}

	expectedLanguages := []Language{
		LangGo, LangJava, LangPython, LangJavaScript,
		LangPHP, LangRuby, LangCSharp, LangSwift,
		LangKotlin,
	}

	for _, lang := range expectedLanguages {
		if prefixes, exists := options.LoggingPrefixes[lang]; !exists {
			t.Errorf("Missing logging prefixes for language: %s", lang)
		} else if len(prefixes) == 0 {
			t.Errorf("Empty logging prefixes for language: %s", lang)
		}
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()

	if len(langs) == 0 {
		t.Error("GetSupportedLanguages() returned empty list")
	}

	// Check for required languages
	required := map[Language]bool{
		LangGo:         false,
		LangJava:       false,
		LangPython:     false,
		LangJavaScript: false,
		LangTypeScript: false,
		LangPHP:        false,
		LangRuby:       false,
		LangCSharp:     false,
	}

	for _, lang := range langs {
		if _, ok := required[lang]; ok {
			required[lang] = true
		}
	}

	for lang, found := range required {
		if !found {
			t.Errorf("Required language %s not found in supported languages", lang)
		}
	}
}

func TestNewCleanerValidation(t *testing.T) {
	tests := []struct {
		name        string
		lang        Language
		options     *CleanerOptions
		shouldError bool
	}{
		{
			name:        "nil options",
			lang:        LangGo,
			options:     nil,
			shouldError: true,
		},
		{
			name:        "invalid language",
			lang:        "invalid",
			options:     DefaultOptions(),
			shouldError: true,
		},
		{
			name:        "valid configuration",
			lang:        LangGo,
			options:     DefaultOptions(),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCleaner(tt.lang, tt.options)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
