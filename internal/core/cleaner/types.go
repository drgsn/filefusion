package cleaner

// Language represents supported programming languages
type Language string

const (
	LangGo         Language = "go"
	LangJava       Language = "java"
	LangPython     Language = "python"
	LangSwift      Language = "swift"
	LangKotlin     Language = "kotlin"
	LangSQL        Language = "sql"
	LangHTML       Language = "html"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangCSS        Language = "css"
	LangCPP        Language = "cpp"
	LangCSharp     Language = "csharp"
	LangPHP        Language = "php"
	LangRuby       Language = "ruby"
	LangBash       Language = "bash"
)

// GetSupportedLanguages returns a list of all supported languages
func GetSupportedLanguages() []Language {
	return []Language{
		LangGo, LangJava, LangPython, LangSwift, LangKotlin,
		LangSQL, LangHTML, LangJavaScript, LangTypeScript, LangCSS,
		LangCPP, LangCSharp, LangPHP, LangRuby, LangBash,
	}
}
