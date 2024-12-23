import { Alert, AlertDescription } from '@/components/ui/alert';
import { FileCode, FileJson, Moon, Sun, Terminal } from 'lucide-react';
import React from 'react';

const HomePage = () => {
  const [isDark, setIsDark] = React.useState(true);

  React.useEffect(() => {
    // Apply dark mode class to document
    document.documentElement.classList.toggle('dark', isDark);
  }, [isDark]);

  return (
    <div className={`min-h-screen ${isDark ? 'dark bg-gray-950' : 'bg-gray-50'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-gray-900' : 'bg-white'} shadow-sm`}>
        <div className="max-w-5xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>FileFusion</h1>
            <div className="flex items-center space-x-6">
              <button 
                onClick={() => setIsDark(!isDark)} 
                className={`p-2 rounded-lg ${isDark ? 'text-gray-300 hover:text-white' : 'text-gray-600 hover:text-gray-900'}`}
              >
                {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
              </button>
              <div className="flex space-x-4">
                <a href="https://github.com/drgsn/filefusion" 
                   className={`${isDark ? 'text-gray-300 hover:text-white' : 'text-gray-600 hover:text-gray-900'}`}>
                  Documentation
                </a>
                <a href="https://github.com/drgsn/filefusion" 
                   className={`${isDark ? 'text-gray-300 hover:text-white' : 'text-gray-600 hover:text-gray-900'}`}>
                  GitHub
                </a>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <div className={`${isDark ? 'bg-gray-900' : 'bg-white'} border-b border-gray-800`}>
        <div className="max-w-5xl mx-auto px-4 py-16">
          <div className="text-center">
            <h2 className={`text-4xl font-bold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              File Concatenation Tool Optimized for LLM Usage
            </h2>
            <p className={`text-xl mb-8 ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
              Concatenate files into formats optimized for Large Language Models while preserving metadata
            </p>
            <div className="flex justify-center space-x-4">
              <a href="https://github.com/drgsn/filefusion/releases" 
                 className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700">
                Download
              </a>
              <a href="https://github.com/drgsn/filefusion" 
                 className={`${isDark ? 'bg-gray-800 text-gray-300' : 'bg-gray-100 text-gray-700'} px-6 py-2 rounded-lg hover:bg-gray-700 hover:text-white`}>
                View on GitHub
              </a>
            </div>
          </div>
        </div>
      </div>

      {/* Features Section */}
      <div className="max-w-5xl mx-auto px-4 py-16">
        <div className="grid md:grid-cols-3 gap-8">
          <div className={`${isDark ? 'bg-gray-900' : 'bg-white'} p-6 rounded-lg shadow-sm`}>
            <FileJson className="w-12 h-12 text-blue-500 mb-4" />
            <h3 className={`text-xl font-semibold mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>Multiple Output Formats</h3>
            <p className={`${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
              Export your concatenated files in XML, JSON, or YAML formats
            </p>
          </div>
          <div className={`${isDark ? 'bg-gray-900' : 'bg-white'} p-6 rounded-lg shadow-sm`}>
            <FileCode className="w-12 h-12 text-blue-500 mb-4" />
            <h3 className={`text-xl font-semibold mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>Pattern Matching</h3>
            <p className={`${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
              Use glob patterns to include or exclude files
            </p>
          </div>
          <div className={`${isDark ? 'bg-gray-900' : 'bg-white'} p-6 rounded-lg shadow-sm`}>
            <Terminal className="w-12 h-12 text-blue-500 mb-4" />
            <h3 className={`text-xl font-semibold mb-2 ${isDark ? 'text-white' : 'text-gray-900'}`}>CLI Interface</h3>
            <p className={`${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
              Simple command-line interface for easy integration
            </p>
          </div>
        </div>
      </div>

      {/* Examples Section */}
      <div className={`${isDark ? 'bg-gray-900' : 'bg-white'} border-t border-gray-800`}>
        <div className="max-w-5xl mx-auto px-4 py-16">
          <h2 className={`text-3xl font-bold mb-8 ${isDark ? 'text-white' : 'text-gray-900'}`}>Examples</h2>
          
          <div className="space-y-6">
            <Alert className={isDark ? 'bg-gray-800 border-gray-700' : ''}>
              <AlertDescription>
                <p className={`font-mono mb-2 ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                  # Basic usage - process all Go files in current directory
                </p>
                <p className={`font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  filefusion --pattern "*.go" --output result.xml .
                </p>
              </AlertDescription>
            </Alert>

            <Alert className={isDark ? 'bg-gray-800 border-gray-700' : ''}>
              <AlertDescription>
                <p className={`font-mono mb-2 ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                  # Multiple patterns and exclusions
                </p>
                <p className={`font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  filefusion --pattern "*.go,*.json" --exclude "vendor/**,*.test.go" --output api.json ./api
                </p>
              </AlertDescription>
            </Alert>

            <Alert className={isDark ? 'bg-gray-800 border-gray-700' : ''}>
              <AlertDescription>
                <p className={`font-mono mb-2 ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                  # Size limits and YAML output
                </p>
                <p className={`font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  filefusion --max-file-size 5MB --max-output-size 20MB --output docs.yaml ./docs
                </p>
              </AlertDescription>
            </Alert>

            <Alert className={isDark ? 'bg-gray-800 border-gray-700' : ''}>
              <AlertDescription>
                <p className={`font-mono mb-2 ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>
                  # Process multiple directories
                </p>
                <p className={`font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  filefusion --pattern "*.go" ./cmd ./internal ./pkg
                </p>
              </AlertDescription>
            </Alert>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="bg-gray-900 text-white">
        <div className="max-w-5xl mx-auto px-4 py-8">
          <p className="text-center text-gray-400">
            FileFusion is open source software licensed under Mozilla Public License Version 2.0
          </p>
        </div>
      </footer>
    </div>
  );
};

export default HomePage;