# HAR Analyzer

HAR Analyzer is a command-line tool for analyzing HTTP Archive (HAR) files with advanced filtering capabilities. It allows you to process large HAR files, apply complex filters, and generate insightful summaries of the data.

## Features

- Read and parse HAR files
- Advanced filtering options (URL, HTTP method, status code, response time, response size, content type)
- Custom query language for complex filtering
- Sort entries by time, size, status code, or URL
- Break down entries into configurable chunks
- Generate summaries for each chunk
- Optional AI analysis using the Anthropic API
- Multiple output formats (text, JSON, CSV)

## Installation

1. Ensure you have Go 1.20 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/yourusername/haranalyzer.git
   ```
3. Change