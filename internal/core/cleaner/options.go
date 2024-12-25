package cleaner

// CleanerOptions defines the configuration options for the code cleaner
type CleanerOptions struct {
	// RemoveComments determines if comments should be removed
	RemoveComments bool

	// PreserveDocComments determines if documentation comments should be preserved
	PreserveDocComments bool

	// RemoveImports determines if import statements should be removed
	RemoveImports bool

	// RemoveLogging determines if logging statements should be removed
	RemoveLogging bool

	// RemoveGettersSetters determines if getter/setter methods should be removed
	RemoveGettersSetters bool

	// OptimizeWhitespace determines if whitespace should be optimized
	OptimizeWhitespace bool

	// RemoveEmptyLines determines if empty lines should be removed
	RemoveEmptyLines bool

	// LoggingPrefixes defines the prefixes of logging statements to remove
	LoggingPrefixes map[Language][]string
}

// DefaultOptions returns a new CleanerOptions with default settings
func DefaultOptions() *CleanerOptions {
	return &CleanerOptions{
		RemoveComments:       true,
		PreserveDocComments:  true,
		RemoveImports:        false,
		RemoveLogging:        true,
		RemoveGettersSetters: true,
		OptimizeWhitespace:   true,
		RemoveEmptyLines:     true,
		LoggingPrefixes: map[Language][]string{
			LangGo:         {"log.", "logger."},
			LangJava:       {"Logger.", "System.out.", "System.err."},
			LangPython:     {"logging.", "logger.", "print(", "print ("},
			LangJavaScript: {"console.", "logger."},
			LangTypeScript: {"console.", "logger."},
			LangPHP:        {"error_log(", "print_r(", "var_dump("},
			LangRuby:       {"puts ", "print ", "p ", "logger."},
			LangCSharp:     {"Console.", "Debug.", "Logger."},
			LangSwift:      {"print(", "debugPrint(", "NSLog("},
			LangKotlin:     {"println(", "print(", "Logger."},
		},
	}
}
