// templr playground - Autocomplete functionality for template editor

// Template functions (Sprig + custom templr functions)
const TEMPLATE_FUNCTIONS = [
  // Helm-like functions
  { name: 'include', signature: 'include "templateName" .', desc: 'Execute a named template and return its output' },
  { name: 'required', signature: 'required "message" .value', desc: 'Fail rendering if value is nil or empty' },
  { name: 'fail', signature: 'fail "message"', desc: 'Explicitly fail rendering with a message' },
  { name: 'safe', signature: 'safe .value "fallback"', desc: 'Render a value or fallback when missing/empty' },

  // YAML/JSON
  { name: 'toYaml', signature: 'toYaml .', desc: 'Serialize data to YAML format' },
  { name: 'fromYaml', signature: 'fromYaml .str', desc: 'Parse YAML string to map' },
  { name: 'mustToYaml', signature: 'mustToYaml .', desc: 'Serialize to YAML (panic on error)' },
  { name: 'mustFromYaml', signature: 'mustFromYaml .str', desc: 'Parse YAML (panic on error)' },
  { name: 'toJson', signature: 'toJson .', desc: 'Serialize data to JSON format' },
  { name: 'fromJson', signature: 'fromJson .str', desc: 'Parse JSON string to map' },

  // TOML (v1.3.0)
  { name: 'toToml', signature: 'toToml .', desc: 'Serialize data to TOML format' },
  { name: 'fromToml', signature: 'fromToml .str', desc: 'Parse TOML string to map' },

  // XML (v1.5.0)
  { name: 'toXml', signature: 'toXml .', desc: 'Serialize data to XML format' },
  { name: 'fromXml', signature: 'fromXml .str', desc: 'Parse XML string to map' },

  // Humanization (v1.3.0)
  { name: 'humanizeBytes', signature: 'humanizeBytes 1048576', desc: 'Format bytes as human-readable size (e.g., "1.0 MB")' },
  { name: 'humanizeNumber', signature: 'humanizeNumber 1234567', desc: 'Add thousand separators (e.g., "1,234,567")' },
  { name: 'humanizeTime', signature: 'humanizeTime .timestamp', desc: 'Relative time format (e.g., "10 months ago")' },
  { name: 'ordinal', signature: 'ordinal 21', desc: 'Convert number to ordinal (e.g., "21st")' },

  // Path functions (v1.3.0)
  { name: 'pathExt', signature: 'pathExt "file.txt"', desc: 'Get file extension (e.g., ".txt")' },
  { name: 'pathStem', signature: 'pathStem "doc.pdf"', desc: 'Get filename without extension (e.g., "doc")' },
  { name: 'pathNormalize', signature: 'pathNormalize "a/b/../c"', desc: 'Normalize path separators (e.g., "a/c")' },
  { name: 'mimeType', signature: 'mimeType "data.json"', desc: 'Detect MIME type from extension' },

  // Validation functions (v1.3.0)
  { name: 'isEmail', signature: 'isEmail .email', desc: 'Validate email address' },
  { name: 'isURL', signature: 'isURL .url', desc: 'Validate URL' },
  { name: 'isIPv4', signature: 'isIPv4 .ip', desc: 'Check if valid IPv4 address' },
  { name: 'isIPv6', signature: 'isIPv6 .ip', desc: 'Check if valid IPv6 address' },
  { name: 'isUUID', signature: 'isUUID .id', desc: 'Check if valid UUID' },

  // Encoding (v1.4.0)
  { name: 'base64url', signature: 'base64url "hello"', desc: 'URL-safe base64 encode' },
  { name: 'base64urlDecode', signature: 'base64urlDecode .encoded', desc: 'Decode URL-safe base64' },
  { name: 'base32', signature: 'base32 "hello"', desc: 'Base32 encode' },
  { name: 'base32Decode', signature: 'base32Decode .encoded', desc: 'Decode base32' },

  // CSV (v1.4.0)
  { name: 'fromCsv', signature: 'fromCsv .csvStr', desc: 'Parse CSV string to slice of maps' },
  { name: 'csvColumn', signature: 'csvColumn .csv "name"', desc: 'Extract single CSV column as slice' },
  { name: 'toCsv', signature: 'toCsv .data', desc: 'Serialize data to CSV format' },

  // Network (v1.4.0)
  { name: 'cidrContains', signature: 'cidrContains "10.0.0.5" "10.0.0.0/24"', desc: 'Check if IP in CIDR range' },
  { name: 'cidrHosts', signature: 'cidrHosts "10.0.0.0/30"', desc: 'List usable hosts in CIDR' },
  { name: 'ipAdd', signature: 'ipAdd "10.0.0.1" 5', desc: 'IP address arithmetic (e.g., "10.0.0.6")' },
  { name: 'ipVersion', signature: 'ipVersion .ip', desc: 'Detect IP version (4 or 6)' },
  { name: 'ipPrivate', signature: 'ipPrivate .ip', desc: 'Check if private IP address' },

  // Math & Statistics (v1.4.0)
  { name: 'sum', signature: 'sum .numbers', desc: 'Sum of array' },
  { name: 'avg', signature: 'avg .numbers', desc: 'Average of array' },
  { name: 'median', signature: 'median .numbers', desc: 'Median value' },
  { name: 'stddev', signature: 'stddev .numbers', desc: 'Standard deviation' },
  { name: 'percentile', signature: 'percentile .numbers 90', desc: 'Calculate percentile' },
  { name: 'clamp', signature: 'clamp 15 0 10', desc: 'Clamp value to range (e.g., 10)' },
  { name: 'roundTo', signature: 'roundTo 3.14159 2', desc: 'Round to N decimals (e.g., 3.14)' },

  // JSON Querying (v1.5.0)
  { name: 'jsonPath', signature: 'jsonPath .json "users.#.name"', desc: 'Query JSON with gjson path syntax' },
  { name: 'jsonQuery', signature: 'jsonQuery .json "users.#(active==true)"', desc: 'Query JSON, return as array' },
  { name: 'jsonSet', signature: 'jsonSet .json "config.enabled" true', desc: 'Modify JSON at path' },

  // Date Parsing (v1.5.0)
  { name: 'dateParse', signature: 'dateParse "March 15, 2024"', desc: 'Parse any common date format' },
  { name: 'dateAdd', signature: 'dateAdd "2024-01-01" "7 days"', desc: 'Add duration to date' },
  { name: 'dateRange', signature: 'dateRange "2024-01-01" "2024-01-07"', desc: 'Generate inclusive date range' },
  { name: 'workdays', signature: 'workdays "2024-01-01" "2024-01-15"', desc: 'Count business days' },

  // Common Sprig functions
  { name: 'upper', signature: 'upper .str', desc: 'Convert to uppercase' },
  { name: 'lower', signature: 'lower .str', desc: 'Convert to lowercase' },
  { name: 'trim', signature: 'trim .str', desc: 'Remove leading/trailing whitespace' },
  { name: 'trimPrefix', signature: 'trimPrefix "pre" .str', desc: 'Remove prefix from string' },
  { name: 'trimSuffix', signature: 'trimSuffix "suf" .str', desc: 'Remove suffix from string' },
  { name: 'replace', signature: 'replace "old" "new" .str', desc: 'Replace all occurrences' },
  { name: 'contains', signature: 'contains "sub" .str', desc: 'Check if string contains substring' },
  { name: 'hasPrefix', signature: 'hasPrefix "pre" .str', desc: 'Check if string has prefix' },
  { name: 'hasSuffix', signature: 'hasSuffix "suf" .str', desc: 'Check if string has suffix' },
  { name: 'split', signature: 'split "," .str', desc: 'Split string by delimiter' },
  { name: 'join', signature: 'join "," .list', desc: 'Join array with delimiter' },
  { name: 'default', signature: 'default "fallback" .value', desc: 'Use fallback if value is empty' },
  { name: 'empty', signature: 'empty .value', desc: 'Check if value is empty' },
  { name: 'coalesce', signature: 'coalesce .a .b .c', desc: 'Return first non-empty value' },
  { name: 'ternary', signature: 'ternary "yes" "no" .condition', desc: 'Ternary operator' },
  { name: 'list', signature: 'list 1 2 3', desc: 'Create a list' },
  { name: 'dict', signature: 'dict "key" "value"', desc: 'Create a dictionary' },
  { name: 'get', signature: 'get .dict "key"', desc: 'Get value from dict' },
  { name: 'set', signature: 'set .dict "key" "value"', desc: 'Set value in dict and return it' },
  { name: 'setd', signature: 'setd . "a.b.c" "value"', desc: 'Set dotted key path in dict' },
  { name: 'unset', signature: 'unset .dict "key"', desc: 'Remove key from dict' },
  { name: 'keys', signature: 'keys .dict', desc: 'Get all keys from dict' },
  { name: 'values', signature: 'values .dict', desc: 'Get all values from dict' },
  { name: 'pick', signature: 'pick .dict "key1" "key2"', desc: 'Select specific keys from dict' },
  { name: 'omit', signature: 'omit .dict "key1" "key2"', desc: 'Remove specific keys from dict' },
  { name: 'merge', signature: 'merge .dict1 .dict2', desc: 'Merge two dicts (shallow)' },
  { name: 'mergeDeep', signature: 'mergeDeep .dict1 .dict2', desc: 'Deep merge two dicts' },
  { name: 'append', signature: 'append .list "item"', desc: 'Append item to list' },
  { name: 'prepend', signature: 'prepend .list "item"', desc: 'Prepend item to list' },
  { name: 'first', signature: 'first .list', desc: 'Get first item' },
  { name: 'last', signature: 'last .list', desc: 'Get last item' },
  { name: 'rest', signature: 'rest .list', desc: 'Get all but first item' },
  { name: 'initial', signature: 'initial .list', desc: 'Get all but last item' },
  { name: 'reverse', signature: 'reverse .list', desc: 'Reverse list' },
  { name: 'uniq', signature: 'uniq .list', desc: 'Remove duplicates from list' },
  { name: 'sortAlpha', signature: 'sortAlpha .list', desc: 'Sort list alphabetically' },
  { name: 'add', signature: 'add 1 2', desc: 'Add numbers' },
  { name: 'sub', signature: 'sub 5 3', desc: 'Subtract numbers' },
  { name: 'mul', signature: 'mul 2 3', desc: 'Multiply numbers' },
  { name: 'div', signature: 'div 10 2', desc: 'Divide numbers' },
  { name: 'mod', signature: 'mod 10 3', desc: 'Modulo operation' },
  { name: 'max', signature: 'max 1 2 3', desc: 'Maximum value' },
  { name: 'min', signature: 'min 1 2 3', desc: 'Minimum value' },
  { name: 'floor', signature: 'floor 3.7', desc: 'Round down' },
  { name: 'ceil', signature: 'ceil 3.2', desc: 'Round up' },
  { name: 'round', signature: 'round 3.5', desc: 'Round to nearest' },
  { name: 'now', signature: 'now', desc: 'Current time' },
  { name: 'date', signature: 'date "2006-01-02" .time', desc: 'Format date/time' },
  { name: 'dateModify', signature: 'dateModify "+24h" .time', desc: 'Modify date/time' },
  { name: 'durationRound', signature: 'durationRound .duration', desc: 'Round duration' },
  { name: 'b64enc', signature: 'b64enc .str', desc: 'Base64 encode' },
  { name: 'b64dec', signature: 'b64dec .str', desc: 'Base64 decode' },
  { name: 'sha256sum', signature: 'sha256sum .str', desc: 'SHA256 hash' },
  { name: 'quote', signature: 'quote .str', desc: 'Quote string' },
  { name: 'squote', signature: 'squote .str', desc: 'Single quote string' },
  { name: 'indent', signature: 'indent 2 .str', desc: 'Indent text' },
  { name: 'nindent', signature: 'nindent 2 .str', desc: 'Newline + indent text' },
  { name: 'trunc', signature: 'trunc 10 .str', desc: 'Truncate string' },
  { name: 'abbrev', signature: 'abbrev 10 .str', desc: 'Abbreviate string' },
  { name: 'randAlphaNum', signature: 'randAlphaNum 10', desc: 'Random alphanumeric string' },
  { name: 'randAlpha', signature: 'randAlpha 10', desc: 'Random alphabetic string' },
  { name: 'randNumeric', signature: 'randNumeric 10', desc: 'Random numeric string' },
  { name: 'uuidv4', signature: 'uuidv4', desc: 'Generate UUIDv4' },
  { name: 'cat', signature: 'cat .str1 .str2', desc: 'Concatenate strings' },
  { name: 'repeat', signature: 'repeat 3 .str', desc: 'Repeat string N times' },
  { name: 'substr', signature: 'substr 0 5 .str', desc: 'Get substring' },
];

const TEMPLATE_KEYWORDS = [
  { name: 'if', signature: 'if .condition', desc: 'Conditional block' },
  { name: 'else', signature: 'else', desc: 'Else block' },
  { name: 'end', signature: 'end', desc: 'End block' },
  { name: 'range', signature: 'range .list', desc: 'Iterate over list' },
  { name: 'with', signature: 'with .value', desc: 'Set context to value' },
  { name: 'define', signature: 'define "templateName"', desc: 'Define a named template' },
  { name: 'template', signature: 'template "templateName" .', desc: 'Execute a template' },
  { name: 'block', signature: 'block "name" .', desc: 'Define a block' },
];

// Variable extractor - parses values.yaml and builds autocomplete suggestions
class VariableExtractor {
  constructor() {
    this.variables = [];
    this.valuesCache = null;
  }

  extractVariables(yamlContent) {
    if (!yamlContent || !yamlContent.trim()) {
      return [];
    }

    try {
      const data = jsyaml.load(yamlContent);
      this.variables = [];
      this.traverse(data, '');
      return this.variables;
    } catch (e) {
      console.warn('Failed to parse values.yaml for autocomplete:', e);
      return [];
    }
  }

  traverse(obj, prefix) {
    if (obj === null || obj === undefined) {
      return;
    }

    for (const key in obj) {
      if (!obj.hasOwnProperty(key)) continue;

      const path = prefix ? `${prefix}.${key}` : key;
      const value = obj[key];
      const type = Array.isArray(value) ? 'array' : typeof value;

      this.variables.push({ path, type, value });

      // Recurse into objects (but not arrays)
      if (typeof value === 'object' && !Array.isArray(value) && value !== null) {
        this.traverse(value, path);
      }
    }
  }

  getNestedSuggestions(dotPath) {
    const parts = dotPath.split('.');
    const searchPrefix = dotPath.endsWith('.') ? dotPath.slice(0, -1) : dotPath;

    // Find variables that start with this path
    const matches = this.variables.filter(v => {
      if (!searchPrefix) return v.path.indexOf('.') === -1; // Top level
      return v.path.startsWith(searchPrefix + '.');
    });

    // Extract the next segment
    const depth = searchPrefix ? searchPrefix.split('.').length : 0;
    const suggestions = new Set();

    matches.forEach(match => {
      const segments = match.path.split('.');
      if (segments.length > depth) {
        suggestions.add(segments[depth]);
      }
    });

    return Array.from(suggestions).map(name => {
      const fullPath = searchPrefix ? `${searchPrefix}.${name}` : name;
      const variable = this.variables.find(v => v.path === fullPath);
      return {
        path: name,
        fullPath: fullPath,
        type: variable ? variable.type : 'unknown',
        value: variable ? variable.value : undefined
      };
    });
  }
}

// Template hint provider - main autocomplete logic
class TemplateHintProvider {
  constructor(app) {
    this.app = app;
    this.variableExtractor = new VariableExtractor();
    this.helperTemplates = [];
  }

  getTemplateHints(editor) {
    const cursor = editor.getCursor();
    const line = editor.getLine(cursor.line);

    // Check if we're inside template delimiters
    const context = this.detectContext(line, cursor.ch);

    if (!context.inTemplate) {
      return null;
    }

    // Get suggestions based on context
    const suggestions = this.getSuggestionsForContext(context);

    if (!suggestions || suggestions.length === 0) {
      return null;
    }

    // Add custom hint function to close braces
    const enhancedSuggestions = suggestions.map(suggestion => ({
      ...suggestion,
      hint: (cm, data, completion) => {
        // Insert the completion text
        cm.replaceRange(completion.text, data.from, data.to);

        // Check if we need to add closing braces
        const newCursor = cm.getCursor();
        const currentLine = cm.getLine(newCursor.line);
        const afterCursor = currentLine.substring(newCursor.ch);

        // If there's no }} after cursor, add it
        if (!afterCursor.trim().startsWith('}}')) {
          cm.replaceRange(' }}', newCursor);
          // Move cursor before the }}
          cm.setCursor({ line: newCursor.line, ch: newCursor.ch });
        }
      }
    }));

    // Return CodeMirror hint object
    return {
      list: enhancedSuggestions,
      from: CodeMirror.Pos(cursor.line, context.start),
      to: CodeMirror.Pos(cursor.line, context.end)
    };
  }

  detectContext(line, cursorCh) {
    const beforeCursor = line.substring(0, cursorCh);
    const afterCursor = line.substring(cursorCh);

    // Find last {{ and first }}
    const lastOpen = beforeCursor.lastIndexOf('{{');
    const nextClose = afterCursor.indexOf('}}');

    // Not in template if no {{ before cursor
    const inTemplate = lastOpen !== -1 && (nextClose !== -1 || afterCursor.indexOf('}}') === -1);

    if (!inTemplate) {
      return { inTemplate: false };
    }

    // Extract the content inside {{ }}
    const templateContent = beforeCursor.substring(lastOpen + 2);

    // Determine context type
    let contextType = 'general';
    let prefix = '';
    let start = cursorCh;
    let end = cursorCh;

    // Check for variable access (starts with .)
    if (templateContent.trim().startsWith('.')) {
      contextType = 'variable';
      const dotMatch = /\.([\w.]*?)$/.exec(templateContent);
      if (dotMatch) {
        prefix = dotMatch[1];
        // Find the position after the last dot
        const lastDotPos = beforeCursor.lastIndexOf('.');
        // If there's content after the dot, replace from after the dot
        // If just a dot with nothing after, start from after the dot
        if (prefix) {
          start = cursorCh - prefix.length;
        } else {
          start = lastDotPos + 1;
        }
      }
    }
    // Check for include function
    else if (/include\s+"([^"]*)"?\s*$/.test(templateContent)) {
      contextType = 'include';
      const match = /include\s+"([^"]*)"?\s*$/.exec(templateContent);
      prefix = match[1] || '';
      start = cursorCh - prefix.length;
    }
    // Check after pipe
    else if (templateContent.includes('|')) {
      contextType = 'pipe';
      // Find the position after the pipe and any whitespace
      const pipePos = templateContent.lastIndexOf('|');
      const afterPipe = templateContent.substring(pipePos + 1);
      const trimmed = afterPipe.trimStart();
      prefix = trimmed;
      // Start position should be after the pipe and whitespace
      start = lastOpen + 2 + pipePos + 1 + (afterPipe.length - trimmed.length);
    }
    // General context
    else {
      // Match word at cursor, but preserve leading whitespace
      const wordMatch = /(\w+)$/.exec(templateContent);
      if (wordMatch) {
        prefix = wordMatch[1];
        start = cursorCh - prefix.length;
      } else {
        // No word yet, start from current position
        start = cursorCh;
      }
    }

    return {
      inTemplate: true,
      contextType,
      prefix,
      start,
      end,
      templateContent
    };
  }

  getSuggestionsForContext(context) {
    const { contextType, prefix, templateContent } = context;
    let suggestions = [];

    switch (contextType) {
      case 'variable':
        suggestions = this.getVariableSuggestions(prefix);
        break;

      case 'include':
        suggestions = this.getHelperTemplateSuggestions(prefix);
        break;

      case 'pipe':
        suggestions = this.getFunctionSuggestions(prefix);
        break;

      case 'general':
      default:
        // Check if we just typed a dot - if so, show variables with dot prefix
        const justTypedDot = templateContent.trim() === '.';

        if (justTypedDot) {
          // Just typed dot, show top-level variables (they need dot prefix)
          suggestions = this.getTopLevelVariableSuggestionsWithDot(prefix);
        } else {
          // All suggestions (functions, keywords, top-level variables)
          suggestions = [
            ...this.getFunctionSuggestions(prefix),
            ...this.getKeywordSuggestions(prefix),
            ...this.getTopLevelVariableSuggestions(prefix)
          ];
        }
        break;
    }

    return suggestions;
  }

  getVariableSuggestions(prefix) {
    this.updateVariablesFromValues();

    const nested = this.variableExtractor.getNestedSuggestions(prefix);

    return nested.map(v => ({
      text: v.path,
      displayText: `${v.path} (${v.type})`,
      className: 'autocomplete-variable',
      type: v.type
    }));
  }

  getFunctionSuggestions(prefix) {
    const lowerPrefix = prefix.toLowerCase();

    return TEMPLATE_FUNCTIONS
      .filter(f => f.name.toLowerCase().startsWith(lowerPrefix))
      .map(f => ({
        text: f.signature,
        displayText: f.signature,
        className: 'autocomplete-function',
        description: f.desc,
        funcData: f
      }));
  }

  getKeywordSuggestions(prefix) {
    const lowerPrefix = prefix.toLowerCase();

    return TEMPLATE_KEYWORDS
      .filter(k => k.name.toLowerCase().startsWith(lowerPrefix))
      .map(k => ({
        text: k.signature,
        displayText: k.signature,
        className: 'autocomplete-keyword',
        description: k.desc
      }));
  }

  getTopLevelVariableSuggestions(prefix) {
    this.updateVariablesFromValues();

    const topLevel = this.variableExtractor.variables
      .filter(v => v.path.indexOf('.') === -1)
      .filter(v => v.path.toLowerCase().startsWith(prefix.toLowerCase()));

    return topLevel.map(v => ({
      text: v.path,
      displayText: `.${v.path} (${v.type})`,
      className: 'autocomplete-variable',
      type: v.type
    }));
  }

  getTopLevelVariableSuggestionsWithDot(prefix) {
    this.updateVariablesFromValues();

    const topLevel = this.variableExtractor.variables
      .filter(v => v.path.indexOf('.') === -1)
      .filter(v => v.path.toLowerCase().startsWith(prefix.toLowerCase()));

    return topLevel.map(v => ({
      text: '.' + v.path,
      displayText: `.${v.path} (${v.type})`,
      className: 'autocomplete-variable',
      type: v.type
    }));
  }

  getHelperTemplateSuggestions(prefix) {
    this.updateHelperTemplates();

    return this.helperTemplates
      .filter(name => name.toLowerCase().includes(prefix.toLowerCase()))
      .map(name => ({
        text: name,
        displayText: name,
        className: 'autocomplete-template'
      }));
  }

  updateVariablesFromValues() {
    const valuesFile = this.app.templateFS.getFile('values.yaml') ||
                        this.app.templateFS.getFile('values.yml');

    if (valuesFile) {
      this.variableExtractor.extractVariables(valuesFile);
    }
  }

  updateHelperTemplates() {
    this.helperTemplates = [];

    for (const [path, content] of this.app.templateFS.files) {
      if (path.includes('_helpers') && path.endsWith('.tpl')) {
        const defineRegex = /\{\{-?\s*define\s+"([^"]+)"\s*-?\}\}/g;
        let match;
        while ((match = defineRegex.exec(content)) !== null) {
          this.helperTemplates.push(match[1]);
        }
      }
    }
  }
}
