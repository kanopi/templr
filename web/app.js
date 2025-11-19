// templr playground - VS Code-like interface
// Virtual file system and editor management

class FileSystem {
  constructor() {
    this.files = new Map(); // path -> content
    this.folders = new Set(); // folder paths
  }

  setFile(path, content) {
    this.files.set(path, content);
    // Add parent folders
    const parts = path.split('/');
    for (let i = 1; i < parts.length; i++) {
      this.folders.add(parts.slice(0, i).join('/'));
    }
  }

  getFile(path) {
    return this.files.get(path) || '';
  }

  deleteFile(path) {
    this.files.delete(path);
  }

  deleteFolder(path) {
    // Delete all files in folder
    for (const [filePath] of this.files) {
      if (filePath.startsWith(path + '/')) {
        this.files.delete(filePath);
      }
    }
    // Delete folder and subfolders
    for (const folder of this.folders) {
      if (folder === path || folder.startsWith(path + '/')) {
        this.folders.delete(folder);
      }
    }
  }

  exists(path) {
    return this.files.has(path) || this.folders.has(path);
  }

  isFolder(path) {
    return this.folders.has(path);
  }

  getTree() {
    const tree = {};
    for (const [path] of this.files) {
      const parts = path.split('/');
      let current = tree;
      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        if (i === parts.length - 1) {
          current[part] = { type: 'file', path };
        } else {
          if (!current[part]) {
            current[part] = { type: 'folder', children: {} };
          }
          current = current[part].children;
        }
      }
    }
    return tree;
  }

  getAllFiles() {
    const result = {};
    for (const [path, content] of this.files) {
      result[path] = content;
    }
    return result;
  }

  clear() {
    this.files.clear();
    this.folders.clear();
  }
}

class PlaygroundApp {
  constructor() {
    this.templateFS = new FileSystem();
    this.outputFS = new FileSystem();
    this.editor = null;
    this.currentFile = null;
    this.selectedItem = null;
    this.openTabs = [];
    this.logCount = 0;
    this.logs = [];
    this.currentTourStep = 1;

    this.initWasm();
  }

  async initWasm() {
    const go = new Go();
    const res = await WebAssembly.instantiateStreaming(fetch('templr.wasm'), go.importObject);
    go.run(res.instance);

    this.initUI();
    this.loadSampleProject();
  }

  initUI() {
    // Initialize CodeMirror editor
    const editorEl = document.getElementById('editor');
    const savedTheme = localStorage.getItem('templr-theme') || 'light';
    const editorTheme = savedTheme === 'light' ? 'default' : 'one-dark';

    this.editor = CodeMirror.fromTextArea(editorEl, {
      lineNumbers: true,
      theme: editorTheme,
      mode: 'yaml',
      lineWrapping: true,
      indentUnit: 2,
      tabSize: 2,
      indentWithTabs: false
    });

    this.editor.on('change', () => {
      if (this.currentFile) {
        this.templateFS.setFile(this.currentFile, this.editor.getValue());
        this.markTabDirty(this.currentFile);
      }
    });

    // Event listeners
    document.getElementById('render').addEventListener('click', () => this.render());
    document.getElementById('loadSample').addEventListener('click', () => this.loadSampleProject());
    document.getElementById('themeSelect').addEventListener('change', (e) => this.setTheme(e.target.value, true));
    document.getElementById('showTour').addEventListener('click', () => this.showTour());
    document.getElementById('uploadZip').addEventListener('click', () => this.uploadProject());
    document.getElementById('downloadTemplates').addEventListener('click', () => this.downloadTemplates());
    document.getElementById('downloadOutput').addEventListener('click', () => this.downloadOutput());
    document.getElementById('newFile').addEventListener('click', () => this.createFile());
    document.getElementById('newFolder').addEventListener('click', () => this.createFolder());
    document.getElementById('uploadFile').addEventListener('click', () => this.uploadFiles());
    document.getElementById('renameItem').addEventListener('click', () => this.renameSelected());
    document.getElementById('deleteItem').addEventListener('click', () => this.deleteSelected());

    // Add keyboard shortcut for rename (F2)
    document.addEventListener('keydown', (e) => {
      if (e.key === 'F2' && this.selectedItem) {
        e.preventDefault();
        this.renameSelected();
      }
    });

    // Hidden file inputs for upload
    document.getElementById('fileUpload').addEventListener('change', (e) => {
      if (e.target.files[0]) {
        this.loadProjectFromZip(e.target.files[0]);
      }
    });

    document.getElementById('singleFileUpload').addEventListener('change', (e) => {
      if (e.target.files.length > 0) {
        this.loadIndividualFiles(e.target.files);
      }
    });

    // Initialize resize handle
    this.initResize();

    // Initialize logging
    this.initLogging();

    // Initialize welcome tour
    this.initTour();

    // Initialize theme
    this.initTheme();

    // Initialize tooltips
    this.initTooltips();
  }

  initLogging() {
    const logHeader = document.getElementById('logHeader');
    const logSection = document.getElementById('logSection');
    const clearLogsBtn = document.getElementById('clearLogs');
    const logCollapseIcon = document.getElementById('logCollapseIcon');

    // Toggle collapse
    logHeader.addEventListener('click', (e) => {
      // Don't toggle if clicking on filters
      if (e.target.closest('.log-filters')) return;

      logSection.classList.toggle('collapsed');
      logCollapseIcon.textContent = logSection.classList.contains('collapsed') ? '▼' : '▲';
    });

    // Clear logs
    clearLogsBtn.addEventListener('click', () => {
      this.clearLogs();
    });

    // Filter toggles
    ['showInfo', 'showWarnings', 'showErrors', 'showDebug'].forEach(id => {
      document.getElementById(id).addEventListener('change', () => {
        this.applyLogFilters();
      });
    });
  }

  log(level, message) {
    const timestamp = new Date().toLocaleTimeString();
    const logEntry = { timestamp, level, message };
    this.logs.push(logEntry);
    this.logCount++;

    const logContent = document.getElementById('logContent');
    const logCount = document.getElementById('logCount');

    // Remove empty message if exists
    const emptyMsg = logContent.querySelector('.log-empty');
    if (emptyMsg) {
      emptyMsg.remove();
    }

    // Create log entry element
    const entry = document.createElement('div');
    entry.className = `log-entry ${level}`;
    entry.innerHTML = `
      <span class="log-timestamp">${timestamp}</span>
      <span class="log-level">${level.toUpperCase()}</span>
      <span class="log-message">${this.escapeHtml(message)}</span>
    `;

    logContent.appendChild(entry);
    logContent.scrollTop = logContent.scrollHeight;

    // Update count
    logCount.textContent = `(${this.logCount})`;

    // Apply filters
    this.applyLogFilters();

    // Expand log section if collapsed and it's an error
    if (level === 'error' || level === 'warning') {
      const logSection = document.getElementById('logSection');
      if (logSection.classList.contains('collapsed')) {
        logSection.classList.remove('collapsed');
        document.getElementById('logCollapseIcon').textContent = '▲';
      }
    }
  }

  logInfo(message) {
    this.log('info', message);
  }

  logWarning(message) {
    this.log('warning', message);
  }

  logError(message) {
    this.log('error', message);
  }

  logDebug(message) {
    const debugMode = document.getElementById('debug').checked;
    if (debugMode) {
      this.log('debug', message);
    }
  }

  clearLogs() {
    this.logs = [];
    this.logCount = 0;
    const logContent = document.getElementById('logContent');
    logContent.innerHTML = '<div class="log-empty">No logs yet. Render templates to see output.</div>';
    document.getElementById('logCount').textContent = '(0)';
  }

  applyLogFilters() {
    const showInfo = document.getElementById('showInfo').checked;
    const showWarnings = document.getElementById('showWarnings').checked;
    const showErrors = document.getElementById('showErrors').checked;
    const showDebug = document.getElementById('showDebug').checked;

    const entries = document.querySelectorAll('.log-entry');
    entries.forEach(entry => {
      const level = entry.classList.contains('info') ? 'info' :
                   entry.classList.contains('warning') ? 'warning' :
                   entry.classList.contains('error') ? 'error' : 'debug';

      const shouldShow =
        (level === 'info' && showInfo) ||
        (level === 'warning' && showWarnings) ||
        (level === 'error' && showErrors) ||
        (level === 'debug' && showDebug);

      entry.classList.toggle('hidden', !shouldShow);
    });
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  getFileIcon(filename) {
    if (filename.endsWith('.tpl')) return '📝';
    if (filename.endsWith('.yaml') || filename.endsWith('.yml')) return '⚙️';
    if (filename.endsWith('.md')) return '📄';
    if (filename.endsWith('.json')) return '📋';
    if (filename.endsWith('.txt')) return '📃';
    return '📄';
  }

  getFolderIcon(collapsed) {
    return collapsed ? '📁' : '📂';
  }

  getFileMode(filename) {
    if (filename.endsWith('.tpl')) return 'go';
    if (filename.endsWith('.yaml') || filename.endsWith('.yml')) return 'yaml';
    if (filename.endsWith('.md')) return 'markdown';
    if (filename.endsWith('.json')) return 'javascript';
    if (filename.endsWith('.js')) return 'javascript';
    return 'yaml';
  }

  renderFileTree(tree, container, isOutput = false) {
    container.innerHTML = '';

    if (Object.keys(tree).length === 0) {
      container.innerHTML = '<div class="empty-state">No files</div>';
      return;
    }

    const renderNode = (node, name, path, parentContainer, level = 0) => {
      const item = document.createElement('div');
      item.className = 'tree-item';
      item.dataset.path = path;
      item.dataset.type = node.type;

      if (node.type === 'folder') {
        item.classList.add('folder');
      }
      item.style.paddingLeft = `${8 + level * 16}px`;

      // Make items draggable (only for template explorer, not output)
      if (!isOutput) {
        item.draggable = true;
        item.addEventListener('dragstart', (e) => this.handleDragStart(e, path, node.type));
        item.addEventListener('dragend', (e) => this.handleDragEnd(e));

        // Allow drop on folders
        if (node.type === 'folder') {
          item.addEventListener('dragover', (e) => this.handleDragOver(e));
          item.addEventListener('drop', (e) => this.handleDrop(e, path));
          item.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        }
      }

      // Add chevron for folders
      if (node.type === 'folder') {
        const chevron = document.createElement('span');
        chevron.className = 'tree-chevron';
        chevron.textContent = '▼'; // Start expanded
        item.appendChild(chevron);
      }

      const icon = document.createElement('span');
      icon.className = 'tree-icon';
      icon.textContent = node.type === 'folder' ? this.getFolderIcon(false) : this.getFileIcon(name);

      const label = document.createElement('span');
      label.textContent = name;

      item.appendChild(icon);
      item.appendChild(label);

      if (!isOutput) {
        item.addEventListener('click', (e) => {
          e.stopPropagation();
          if (node.type === 'file') {
            this.openFile(path);
          } else {
            // Toggle folder
            const children = item.nextElementSibling;
            const chevron = item.querySelector('.tree-chevron');
            if (children && children.classList.contains('tree-children')) {
              const isCollapsed = children.classList.contains('collapsed');
              children.classList.toggle('collapsed');
              icon.textContent = !isCollapsed
                ? this.getFolderIcon(true)
                : this.getFolderIcon(false);
              if (chevron) {
                chevron.textContent = !isCollapsed ? '▶' : '▼';
              }
            }
          }
          this.selectedItem = path;
          // Update selection
          document.getElementById('templateExplorer').querySelectorAll('.tree-item').forEach(el => el.classList.remove('selected'));
          item.classList.add('selected');
        });
      } else {
        // Output tree - click to view (read-only)
        if (node.type === 'file') {
          item.addEventListener('click', (e) => {
            e.stopPropagation();
            this.viewOutputFile(path);
          });
        }
      }

      parentContainer.appendChild(item);

      if (node.type === 'folder' && node.children) {
        const childrenContainer = document.createElement('div');
        childrenContainer.className = 'tree-children';

        const entries = Object.entries(node.children).sort((a, b) => {
          // Folders first
          if (a[1].type !== b[1].type) {
            return a[1].type === 'folder' ? -1 : 1;
          }
          return a[0].localeCompare(b[0]);
        });

        for (const [childName, childNode] of entries) {
          const childPath = path ? `${path}/${childName}` : childName;
          renderNode(childNode, childName, childPath, childrenContainer, level + 1);
        }

        parentContainer.appendChild(childrenContainer);
      }
    };

    const entries = Object.entries(tree).sort((a, b) => {
      if (a[1].type !== b[1].type) {
        return a[1].type === 'folder' ? -1 : 1;
      }
      return a[0].localeCompare(b[0]);
    });

    for (const [name, node] of entries) {
      renderNode(node, name, name, container, 0);
    }
  }

  updateTemplateExplorer() {
    const tree = this.templateFS.getTree();
    const explorer = document.getElementById('templateExplorer');
    this.renderFileTree(tree, explorer, false);

    // Allow dropping files at root level
    explorer.addEventListener('dragover', (e) => {
      // Only handle if dragging over empty space (not over a tree-item)
      if (e.target === explorer || e.target.classList.contains('empty-state')) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
      }
    });

    explorer.addEventListener('drop', (e) => {
      // Only handle if dropping on empty space
      if (e.target === explorer || e.target.classList.contains('empty-state')) {
        e.preventDefault();
        if (this.draggedItem) {
          this.handleDrop(e, ''); // Empty string = root level
        }
      }
    });
  }

  updateOutputExplorer() {
    const tree = this.outputFS.getTree();
    this.renderFileTree(tree, document.getElementById('outputExplorer'), true);
  }

  openFile(path) {
    const content = this.templateFS.getFile(path);
    this.currentFile = path;

    // Show editor, hide placeholder
    document.getElementById('editorPlaceholder').classList.add('hidden');
    this.editor.getWrapperElement().style.display = 'block';

    // Set content and mode
    this.editor.setValue(content);
    const mode = this.getFileMode(path);
    this.editor.setOption('mode', mode);

    // Add to tabs
    this.addTab(path);
    this.switchToTab(path);
  }

  viewOutputFile(path) {
    const content = this.outputFS.getFile(path);

    // Show content in read-only editor
    document.getElementById('editorPlaceholder').classList.add('hidden');
    this.editor.getWrapperElement().style.display = 'block';

    this.editor.setValue(content);
    const mode = this.getFileMode(path);
    this.editor.setOption('mode', mode);
    this.editor.setOption('readOnly', true);

    // Update current file
    this.currentFile = null;
    this.switchToTab(null);
  }

  addTab(path) {
    if (!this.openTabs.includes(path)) {
      this.openTabs.push(path);
    }
    this.renderTabs();
  }

  removeTab(path) {
    const index = this.openTabs.indexOf(path);
    if (index > -1) {
      this.openTabs.splice(index, 1);

      // If current file was closed, open another tab or show placeholder
      if (this.currentFile === path) {
        if (this.openTabs.length > 0) {
          this.openFile(this.openTabs[this.openTabs.length - 1]);
        } else {
          this.currentFile = null;
          document.getElementById('editorPlaceholder').classList.remove('hidden');
          this.editor.getWrapperElement().style.display = 'none';
        }
      }
    }
    this.renderTabs();
  }

  switchToTab(path) {
    this.currentFile = path;
    this.editor.setOption('readOnly', false);
    this.renderTabs();
  }

  markTabDirty(path) {
    // Could add visual indicator for unsaved changes
  }

  renderTabs() {
    const tabsContainer = document.getElementById('tabs');
    tabsContainer.innerHTML = '';

    for (const path of this.openTabs) {
      const tab = document.createElement('div');
      tab.className = 'tab';
      if (path === this.currentFile) {
        tab.classList.add('active');
      }

      const icon = document.createElement('span');
      icon.className = 'tree-icon';
      icon.textContent = this.getFileIcon(path);

      const label = document.createElement('span');
      label.textContent = path.split('/').pop();

      const close = document.createElement('span');
      close.className = 'tab-close';
      close.textContent = '×';
      close.addEventListener('click', (e) => {
        e.stopPropagation();
        this.removeTab(path);
      });

      tab.appendChild(icon);
      tab.appendChild(label);
      tab.appendChild(close);

      tab.addEventListener('click', () => {
        this.openFile(path);
      });

      tabsContainer.appendChild(tab);
    }
  }

  async createFile() {
    const parentPath = (this.selectedItem && this.templateFS.isFolder(this.selectedItem))
      ? this.selectedItem
      : '';

    this.addInlineEditor('file', parentPath);
  }

  async createFolder() {
    const parentPath = (this.selectedItem && this.templateFS.isFolder(this.selectedItem))
      ? this.selectedItem
      : '';

    this.addInlineEditor('folder', parentPath);
  }

  addInlineEditor(type, parentPath) {
    // Remove any existing inline editor
    const existingEditor = document.querySelector('.tree-item.editing');
    if (existingEditor) {
      existingEditor.remove();
    }

    const explorer = document.getElementById('templateExplorer');

    // Create inline editor element
    const editorItem = document.createElement('div');
    editorItem.className = 'tree-item editing';

    // Calculate indent level
    const level = parentPath ? parentPath.split('/').length : 0;
    editorItem.style.paddingLeft = `${8 + level * 16}px`;

    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    icon.textContent = type === 'folder' ? '📁' : '📄';

    const input = document.createElement('input');
    input.className = 'tree-item-input';
    input.type = 'text';
    input.placeholder = type === 'folder' ? 'Folder name' : 'File name (e.g., template.tpl)';

    editorItem.appendChild(icon);
    editorItem.appendChild(input);

    // Insert at the right position
    if (parentPath) {
      // Find the parent folder's children container
      const items = explorer.querySelectorAll('.tree-item');
      let insertPoint = null;

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        const itemPath = item.dataset.path;

        if (itemPath === parentPath) {
          // Insert after parent, before its children container
          const childrenContainer = item.nextElementSibling;
          if (childrenContainer && childrenContainer.classList.contains('tree-children')) {
            insertPoint = childrenContainer;
            break;
          }
        }
      }

      if (insertPoint) {
        insertPoint.insertBefore(editorItem, insertPoint.firstChild);
      } else {
        explorer.insertBefore(editorItem, explorer.firstChild);
      }
    } else {
      // Insert at top level
      explorer.insertBefore(editorItem, explorer.firstChild);
    }

    // Focus input
    input.focus();

    // Handle confirm/cancel
    const confirm = () => {
      const name = input.value.trim();
      if (!name) {
        editorItem.remove();
        return;
      }

      const fullPath = parentPath ? `${parentPath}/${name}` : name;

      if (this.templateFS.exists(fullPath)) {
        this.logError(`${type === 'folder' ? 'Folder' : 'File'} already exists: ${fullPath}`);
        input.focus();
        input.select();
        return;
      }

      if (type === 'folder') {
        this.templateFS.folders.add(fullPath);
        this.logInfo(`Created folder: ${fullPath}`);
      } else {
        this.templateFS.setFile(fullPath, '');
        this.logInfo(`Created file: ${fullPath}`);
      }

      editorItem.remove();
      this.updateTemplateExplorer();

      if (type === 'file') {
        this.openFile(fullPath);
      }
    };

    const cancel = () => {
      editorItem.remove();
    };

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        confirm();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        cancel();
      }
    });

    input.addEventListener('blur', () => {
      // Delay to allow click events to process
      setTimeout(cancel, 150);
    });
  }

  renameSelected() {
    if (!this.selectedItem) {
      this.logWarning('No item selected to rename');
      return;
    }

    const oldPath = this.selectedItem;
    const isFolder = this.templateFS.isFolder(oldPath);
    const parts = oldPath.split('/');
    const oldName = parts[parts.length - 1];
    const parentPath = parts.slice(0, -1).join('/');

    // Find the tree item in the explorer
    const explorer = document.getElementById('templateExplorer');
    const treeItem = Array.from(explorer.querySelectorAll('.tree-item')).find(
      el => el.dataset.path === oldPath
    );

    if (!treeItem) {
      this.logError('Could not find item in tree');
      return;
    }

    // Remove existing inline editors
    const existingEditor = document.querySelector('.tree-item.editing');
    if (existingEditor) {
      existingEditor.remove();
    }

    // Replace the label with an input
    const label = treeItem.querySelector('span:last-child');
    if (!label) return;

    const input = document.createElement('input');
    input.className = 'tree-item-input';
    input.type = 'text';
    input.value = oldName;
    input.style.flex = '1';
    input.style.minWidth = '100px';

    // Replace label with input
    label.replaceWith(input);
    treeItem.classList.add('editing');

    // Select the filename without extension for files
    input.focus();
    if (!isFolder && oldName.includes('.')) {
      const dotIndex = oldName.lastIndexOf('.');
      input.setSelectionRange(0, dotIndex);
    } else {
      input.select();
    }

    const confirm = () => {
      const newName = input.value.trim();

      if (!newName) {
        this.logError('Name cannot be empty');
        input.focus();
        input.select();
        return;
      }

      if (newName === oldName) {
        // No change, just cancel
        treeItem.classList.remove('editing');
        input.replaceWith(label);
        return;
      }

      const newPath = parentPath ? `${parentPath}/${newName}` : newName;

      if (this.templateFS.exists(newPath)) {
        this.logError(`Item already exists: ${newPath}`);
        input.focus();
        input.select();
        return;
      }

      // Perform the rename
      if (isFolder) {
        this.renameFolder(oldPath, newPath);
      } else {
        this.renameFile(oldPath, newPath);
      }

      this.selectedItem = newPath;
    };

    const cancel = () => {
      treeItem.classList.remove('editing');
      input.replaceWith(label);
    };

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        confirm();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        cancel();
      }
    });

    input.addEventListener('blur', () => {
      setTimeout(cancel, 150);
    });
  }

  renameFile(oldPath, newPath) {
    const content = this.templateFS.getFile(oldPath);
    this.templateFS.setFile(newPath, content);
    this.templateFS.deleteFile(oldPath);

    // Update tabs if file is open
    const tabIndex = this.openTabs.indexOf(oldPath);
    if (tabIndex !== -1) {
      this.openTabs[tabIndex] = newPath;
      if (this.currentFile === oldPath) {
        this.currentFile = newPath;
      }
    }

    this.updateTemplateExplorer();
    this.renderTabs();
    this.logInfo(`Renamed file: ${oldPath} → ${newPath}`);
  }

  renameFolder(oldPath, newPath) {
    // Get all files in the folder
    const filesToRename = [];
    for (const [filePath] of this.templateFS.files) {
      if (filePath.startsWith(oldPath + '/')) {
        filesToRename.push(filePath);
      }
    }

    // Rename all files
    for (const filePath of filesToRename) {
      const relativePath = filePath.substring(oldPath.length);
      const newFilePath = newPath + relativePath;
      const content = this.templateFS.getFile(filePath);
      this.templateFS.setFile(newFilePath, content);

      // Update tabs if file is open
      const tabIndex = this.openTabs.indexOf(filePath);
      if (tabIndex !== -1) {
        this.openTabs[tabIndex] = newFilePath;
        if (this.currentFile === filePath) {
          this.currentFile = newFilePath;
        }
      }
    }

    // Delete old folder
    this.templateFS.deleteFolder(oldPath);

    // Add new folder explicitly
    this.templateFS.folders.add(newPath);

    this.updateTemplateExplorer();
    this.renderTabs();
    this.logInfo(`Renamed folder: ${oldPath} → ${newPath} (${filesToRename.length} items)`);
  }

  deleteSelected() {
    if (!this.selectedItem) {
      this.logWarning('No item selected to delete');
      return;
    }

    if (!confirm(`Delete ${this.selectedItem}?`)) return;

    if (this.templateFS.isFolder(this.selectedItem)) {
      this.templateFS.deleteFolder(this.selectedItem);
      this.logInfo(`Deleted folder: ${this.selectedItem}`);
    } else {
      this.templateFS.deleteFile(this.selectedItem);
      this.removeTab(this.selectedItem);
      this.logInfo(`Deleted file: ${this.selectedItem}`);
    }

    this.selectedItem = null;
    this.updateTemplateExplorer();
  }

  async loadProjectFromZip(file) {
    try {
      const JSZip = (await import('https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js')).default;
      const data = await file.arrayBuffer();
      const zip = await JSZip.loadAsync(data);

      this.templateFS.clear();
      this.openTabs = [];

      const promises = [];
      zip.forEach((path, entry) => {
        if (!entry.dir) {
          promises.push(entry.async('string').then(content => {
            this.templateFS.setFile(path, content);
          }));
        }
      });

      await Promise.all(promises);
      this.updateTemplateExplorer();

      // Open first file
      const firstFile = Array.from(this.templateFS.files.keys())[0];
      if (firstFile) {
        this.openFile(firstFile);
      }
    } catch (e) {
      alert(`Error loading zip: ${e.message}`);
    }
  }

  uploadProject() {
    document.getElementById('fileUpload').click();
  }

  async downloadTemplates() {
    const JSZip = (await import('https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js')).default;
    const zip = new JSZip();

    for (const [path, content] of this.templateFS.files) {
      zip.file(path, content);
    }

    const blob = await zip.generateAsync({ type: 'blob' });
    this.downloadBlob(blob, 'templates.zip');
  }

  async downloadOutput() {
    if (this.outputFS.files.size === 0) {
      alert('No output files. Click "Render" first.');
      return;
    }

    const JSZip = (await import('https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js')).default;
    const zip = new JSZip();

    for (const [path, content] of this.outputFS.files) {
      zip.file(path, content);
    }

    const blob = await zip.generateAsync({ type: 'blob' });
    this.downloadBlob(blob, 'output.zip');
  }

  downloadBlob(blob, filename) {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  async render() {
    this.logInfo('Starting render process...');

    // Get options
    const strict = document.getElementById('strict').checked;
    const defaultMissing = document.getElementById('defaultMissing').value || '<no value>';
    const injectGuard = document.getElementById('injectGuard').checked;
    const guardMarker = document.getElementById('guardMarker').value || '#templr generated';
    const extensionsInput = document.getElementById('extensions').value || 'tpl';

    // Parse extensions (comma-separated, trim whitespace, add dot prefix)
    const extensions = extensionsInput.split(',').map(ext => {
      ext = ext.trim();
      return ext.startsWith('.') ? ext : `.${ext}`;
    });

    this.logDebug(`Options: strict=${strict}, defaultMissing="${defaultMissing}", injectGuard=${injectGuard}, extensions=${extensions.join(',')}`);

    // Get all template files
    const files = this.templateFS.getAllFiles();

    if (Object.keys(files).length === 0) {
      this.logError('No template files to render');
      return;
    }

    this.logDebug(`Found ${Object.keys(files).length} files in template directory`);

    // Find values file
    let valuesContent = '';
    for (const [path, content] of Object.entries(files)) {
      if (path === 'values.yaml' || path === 'values.yml') {
        valuesContent = content;
        this.logDebug(`Using values file: ${path}`);
        break;
      }
    }

    if (!valuesContent) {
      this.logWarning('No values.yaml found, rendering with empty values');
    }

    // Clear output
    this.outputFS.clear();

    // Helper function to check if file matches extensions
    const hasTemplateExtension = (path) => {
      return extensions.some(ext => path.endsWith(ext));
    };

    // Count template files
    const templateFiles = Object.entries(files).filter(([path]) =>
      hasTemplateExtension(path) && !path.includes('_helpers')
    );

    this.logInfo(`Rendering ${templateFiles.length} template(s) with extensions: ${extensions.join(', ')}...`);

    // Render each template file
    const errors = [];
    const successes = [];

    for (const [path, content] of Object.entries(files)) {
      if (!hasTemplateExtension(path)) continue;

      // Skip helper files
      if (path.includes('_helpers')) {
        this.logDebug(`Skipping helper file: ${path}`);
        continue;
      }

      this.logDebug(`Rendering template: ${path}`);

      // Find helpers (look for _helpers with any template extension)
      let helpers = '';
      for (const [hPath, hContent] of Object.entries(files)) {
        if (hPath.includes('_helpers') && hasTemplateExtension(hPath)) {
          helpers += hContent + '\n';
        }
      }

      const payload = JSON.stringify({
        template: content,
        values: valuesContent,
        helpers: helpers,
        defaultMissing,
        strict,
        files: {},
        injectGuard,
        guardMarker
      });

      try {
        const res = JSON.parse(window.templrRender(payload));
        if (res.error) {
          errors.push(`${path}: ${res.error}`);
          this.logError(`Failed to render ${path}: ${res.error}`);
        } else {
          // Remove template extension for output
          let outPath = path;
          for (const ext of extensions) {
            if (path.endsWith(ext)) {
              outPath = path.slice(0, -ext.length);
              break;
            }
          }
          this.outputFS.setFile(outPath, res.output);
          successes.push(path);
          this.logDebug(`Successfully rendered ${path} → ${outPath}`);
        }
      } catch (e) {
        errors.push(`${path}: ${e.message}`);
        this.logError(`Exception rendering ${path}: ${e.message}`);
      }
    }

    this.updateOutputExplorer();

    // Log final results
    if (errors.length > 0) {
      this.logWarning(`Render completed with ${errors.length} error(s), ${successes.length} success(es)`);
      errors.forEach(err => this.logError(err));
    } else {
      this.logInfo(`✓ Successfully rendered ${this.outputFS.files.size} file(s)`);
    }
  }

  loadSampleProject() {
    this.templateFS.clear();
    this.openTabs = [];

    // Sample template files
    this.templateFS.setFile('values.yaml', `project:
  name: "templr playground"
  description: "A VS Code-like web interface for testing Go templates"
  features:
    - "Live template rendering"
    - "File management (create, rename, delete, drag & drop)"
    - "Multiple file extensions support"
    - "Helper templates with 'include' function"
    - "Collapsible directory tree"
    - "Syntax highlighting"

author:
  name: "Your Name"
  email: "you@example.com"

version: "1.0.0"
date: "2024-01-15"`);

    this.templateFS.setFile('_helpers.tpl', `{{- define "project.title" -}}
# {{ .project.name }}
{{- end -}}

{{- define "project.badges" -}}
![Version](https://img.shields.io/badge/version-{{ .version }}-blue)
![Go Templates](https://img.shields.io/badge/go-templates-00ADD8)
{{- end -}}

{{- define "feature.list" -}}
{{- range .project.features }}
- ✓ {{ . }}
{{- end }}
{{- end -}}`);

    this.templateFS.setFile('README.md.tpl', `{{ include "project.title" . }}

{{ include "project.badges" . }}

> {{ .project.description }}

## What is templr playground?

This is an interactive web-based playground for **templr**, a powerful Go template rendering tool inspired by Helm. It provides a VS Code-like interface where you can:

{{ include "feature.list" . }}

## Features

### Template System
- **Go Templates**: Full support for Go's \`text/template\` syntax
- **Sprig Functions**: Access to 60+ Sprig template functions
- **Custom Functions**: Additional helpers like \`include\`, \`required\`, \`safe\`, and more
- **Helper Templates**: Define reusable template blocks with \`define\` and call them with \`include\`

### File Management
- Create, rename, and delete files and folders
- Drag and drop files between directories
- Support for multiple file extensions (tpl, yaml, md, txt, json, etc.)
- Collapsible directory tree with visual indicators

### Rendering Options
- **Strict Mode**: Enable \`missingkey=error\` behavior
- **Custom Extensions**: Render any file extension as a template
- **Guard Injection**: Automatically inject guard comments in rendered files
- **Debug Mode**: Verbose logging for troubleshooting

## Quick Start

1. **Edit Values**: Modify \`values.yaml\` to customize your data
2. **Edit Templates**: Create or modify template files (like this \`README.md.tpl\`)
3. **Click Render**: Process all templates and see the output
4. **Download**: Export rendered files or template projects as zip

## Example Usage

This README itself is a template! The values come from \`values.yaml\`:

- Project Name: **{{ .project.name }}**
- Version: **{{ .version }}**
- Author: **{{ .author.name }}** <{{ .author.email }}>
- Date: **{{ .date }}**

## Template Syntax Examples

### Variables
\`\`\`
{{ "{{" }} .project.name {{ "}}" }}  →  {{ .project.name }}
{{ "{{" }} .version {{ "}}" }}       →  {{ .version }}
\`\`\`

### Conditionals
\`\`\`
{{ "{{" }}- if .author.name {{ "}}" }}
Author: {{ "{{" }} .author.name {{ "}}" }}
{{ "{{" }}- end {{ "}}" }}
\`\`\`

### Loops
\`\`\`
{{ "{{" }}- range .project.features {{ "}}" }}
- {{ "{{" }} . {{ "}}" }}
{{ "{{" }}- end {{ "}}" }}
\`\`\`

### Helper Templates
\`\`\`
{{ "{{" }} include "project.title" . {{ "}}" }}
\`\`\`

## Learn More

- [templr Documentation](https://github.com/kanopicode/templr)
- [Go Template Syntax](https://pkg.go.dev/text/template)
- [Sprig Functions](http://masterminds.github.io/sprig/)

---

**Generated with templr playground** • Version {{ .version }}
`);

    this.templateFS.setFile('README.md', `# templr playground

Welcome to the **templr playground**!

## Getting Started

This playground demonstrates templr's template rendering capabilities:

1. **README.md.tpl** - A template that generates this README
2. **values.yaml** - Configuration values used in templates
3. **_helpers.tpl** - Reusable template functions

### Try It Out

1. Click the **Render** button to process templates
2. Check the **Rendered Output** section to see the generated README
3. Modify \`values.yaml\` or \`README.md.tpl\` and render again
4. Create new template files with any extension

### Features to Explore

- **File Management**: Create, rename, delete files using the toolbar buttons
- **Drag & Drop**: Move files between folders
- **Extensions**: Set which file extensions to render (e.g., \`tpl,md,yaml\`)
- **Helpers**: Use \`{{ include "helper.name" . }}\` to call defined templates
- **Logging**: Enable Debug mode to see detailed render information

Happy templating! 🚀
`);

    this.updateTemplateExplorer();
    this.openFile('README.md');
  }

  // Upload individual files
  uploadFiles() {
    document.getElementById('singleFileUpload').click();
  }

  async loadIndividualFiles(files) {
    for (const file of files) {
      try {
        const content = await file.text();
        let path = file.name;

        // If a folder is selected, add files to that folder
        if (this.selectedItem && this.templateFS.isFolder(this.selectedItem)) {
          path = `${this.selectedItem}/${file.name}`;
        }

        this.templateFS.setFile(path, content);
      } catch (e) {
        alert(`Error reading ${file.name}: ${e.message}`);
      }
    }

    this.updateTemplateExplorer();

    // Open first uploaded file
    const firstFile = files[0]?.name;
    if (firstFile) {
      let path = firstFile;
      if (this.selectedItem && this.templateFS.isFolder(this.selectedItem)) {
        path = `${this.selectedItem}/${firstFile}`;
      }
      if (this.templateFS.files.has(path)) {
        this.openFile(path);
      }
    }
  }

  // Initialize sidebar resize
  initResize() {
    const sidebar = document.getElementById('sidebar');
    const resizeHandle = document.getElementById('resizeHandle');
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;

    resizeHandle.addEventListener('mousedown', (e) => {
      isResizing = true;
      startX = e.clientX;
      startWidth = sidebar.offsetWidth;
      resizeHandle.classList.add('resizing');
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
      e.preventDefault();
    });

    document.addEventListener('mousemove', (e) => {
      if (!isResizing) return;

      const delta = e.clientX - startX;
      const newWidth = startWidth + delta;
      const minWidth = 200;
      const maxWidth = window.innerWidth * 0.6;

      if (newWidth >= minWidth && newWidth <= maxWidth) {
        sidebar.style.width = `${newWidth}px`;
      }
    });

    document.addEventListener('mouseup', () => {
      if (isResizing) {
        isResizing = false;
        resizeHandle.classList.remove('resizing');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }
    });
  }

  // Welcome Tour
  initTour() {
    // Check if user has seen the tour before
    const hasSeenTour = localStorage.getItem('templr-tour-completed');

    if (!hasSeenTour) {
      // Show the tour after a brief delay
      setTimeout(() => {
        this.showTour();
      }, 500);
    }

    // Set up tour event listeners
    const modal = document.getElementById('welcomeModal');
    const closeBtn = document.getElementById('closeModal');
    const prevBtn = document.getElementById('tourPrev');
    const nextBtn = document.getElementById('tourNext');
    const finishBtn = document.getElementById('tourFinish');
    const dots = document.querySelectorAll('.tour-dot');

    closeBtn.addEventListener('click', () => this.closeTour());
    prevBtn.addEventListener('click', () => this.prevStep());
    nextBtn.addEventListener('click', () => this.nextStep());
    finishBtn.addEventListener('click', () => this.finishTour());

    // Allow clicking on dots to jump to steps
    dots.forEach(dot => {
      dot.addEventListener('click', () => {
        const step = parseInt(dot.dataset.step);
        this.goToStep(step);
      });
    });

    // Close modal when clicking outside
    modal.addEventListener('click', (e) => {
      if (e.target === modal) {
        this.closeTour();
      }
    });
  }

  showTour() {
    const modal = document.getElementById('welcomeModal');
    modal.classList.add('show');
    this.currentTourStep = 1;
    this.updateTourStep();
  }

  closeTour() {
    const modal = document.getElementById('welcomeModal');
    modal.classList.remove('show');
    // Mark tour as seen when user closes it (even if not completed)
    localStorage.setItem('templr-tour-completed', 'true');
  }

  finishTour() {
    const isFirstTime = !localStorage.getItem('templr-tour-completed');
    localStorage.setItem('templr-tour-completed', 'true');
    this.closeTour();

    // Only load sample project if this is the first time seeing the tour
    if (isFirstTime) {
      this.loadSampleProject();
    }
  }

  goToStep(step) {
    this.currentTourStep = step;
    this.updateTourStep();
  }

  nextStep() {
    if (this.currentTourStep < 4) {
      this.currentTourStep++;
      this.updateTourStep();
    }
  }

  prevStep() {
    if (this.currentTourStep > 1) {
      this.currentTourStep--;
      this.updateTourStep();
    }
  }

  updateTourStep() {
    // Update step visibility
    document.querySelectorAll('.tour-step').forEach(step => {
      step.classList.remove('active');
      if (parseInt(step.dataset.step) === this.currentTourStep) {
        step.classList.add('active');
      }
    });

    // Update dots
    document.querySelectorAll('.tour-dot').forEach(dot => {
      dot.classList.remove('active');
      if (parseInt(dot.dataset.step) === this.currentTourStep) {
        dot.classList.add('active');
      }
    });

    // Update navigation buttons
    const prevBtn = document.getElementById('tourPrev');
    const nextBtn = document.getElementById('tourNext');
    const finishBtn = document.getElementById('tourFinish');

    prevBtn.disabled = this.currentTourStep === 1;

    if (this.currentTourStep === 4) {
      nextBtn.style.display = 'none';
      finishBtn.style.display = 'block';
    } else {
      nextBtn.style.display = 'block';
      finishBtn.style.display = 'none';
    }
  }

  // Theme selector
  initTheme() {
    // Load saved theme preference or default to light
    const savedTheme = localStorage.getItem('templr-theme') || 'light';
    const themeSelect = document.getElementById('themeSelect');
    themeSelect.value = savedTheme;
    this.setTheme(savedTheme, false);
  }

  setTheme(theme, save = true) {
    const body = document.body;
    const themeSelect = document.getElementById('themeSelect');

    if (theme === 'light') {
      body.classList.add('light-theme');

      // Update CodeMirror theme if editor exists
      if (this.editor) {
        this.editor.setOption('theme', 'default');
      }
    } else {
      body.classList.remove('light-theme');

      // Update CodeMirror theme if editor exists
      if (this.editor) {
        this.editor.setOption('theme', 'one-dark');
      }
    }

    // Update dropdown to match
    if (themeSelect.value !== theme) {
      themeSelect.value = theme;
    }

    // Save preference
    if (save) {
      localStorage.setItem('templr-theme', theme);
    }
  }

  // Tooltips
  initTooltips() {
    // Initialize Tippy.js for all elements with data-tippy-content
    tippy('[data-tippy-content]', {
      theme: 'custom',
      placement: 'top',
      arrow: true,
      animation: 'scale',
      duration: [200, 150],
      maxWidth: 350,
    });

    // Also support title attributes (convert to Tippy)
    tippy('[title]', {
      theme: 'custom',
      placement: 'top',
      arrow: true,
      animation: 'scale',
      duration: [200, 150],
      maxWidth: 350,
      content: (reference) => reference.getAttribute('title'),
      onShow(instance) {
        // Remove title to prevent native tooltip
        instance.reference.removeAttribute('title');
      },
      onHidden(instance) {
        // Restore title attribute
        instance.reference.setAttribute('title', instance.props.content);
      }
    });
  }

  // Drag and drop handlers
  handleDragStart(e, path, type) {
    e.stopPropagation();
    this.draggedItem = { path, type };
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', path);
    e.target.style.opacity = '0.5';
    this.logDebug(`Started dragging: ${path} (${type})`);
  }

  handleDragEnd(e) {
    e.target.style.opacity = '1';
  }

  handleDragOver(e) {
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = 'move';

    // Visual feedback - highlight drop zone
    const item = e.currentTarget;
    if (!item.classList.contains('drag-over')) {
      item.classList.add('drag-over');
    }
  }

  handleDragLeave(e) {
    e.stopPropagation();
    const item = e.currentTarget;
    item.classList.remove('drag-over');
  }

  handleDrop(e, targetFolderPath) {
    e.preventDefault();
    e.stopPropagation();

    const item = e.currentTarget;
    item.classList.remove('drag-over');

    if (!this.draggedItem) return;

    const { path: sourcePath, type: sourceType } = this.draggedItem;

    // Prevent dropping into self
    if (sourcePath === targetFolderPath) {
      this.logWarning('Cannot move item into itself');
      this.draggedItem = null;
      return;
    }

    // Prevent dropping folder into its own child
    if (sourceType === 'folder' && targetFolderPath.startsWith(sourcePath + '/')) {
      this.logWarning('Cannot move folder into its own subfolder');
      this.draggedItem = null;
      return;
    }

    // Extract filename/foldername
    const parts = sourcePath.split('/');
    const name = parts[parts.length - 1];
    const newPath = targetFolderPath ? `${targetFolderPath}/${name}` : name;

    // Check if destination already exists
    if (this.templateFS.exists(newPath)) {
      this.logError(`Item already exists at destination: ${newPath}`);
      this.draggedItem = null;
      return;
    }

    // Move the item
    if (sourceType === 'file') {
      this.moveFile(sourcePath, newPath);
    } else if (sourceType === 'folder') {
      this.moveFolder(sourcePath, newPath);
    }

    this.draggedItem = null;
  }

  moveFile(oldPath, newPath) {
    const content = this.templateFS.getFile(oldPath);
    this.templateFS.setFile(newPath, content);
    this.templateFS.deleteFile(oldPath);

    // Update tabs if file is open
    const tabIndex = this.openTabs.indexOf(oldPath);
    if (tabIndex !== -1) {
      this.openTabs[tabIndex] = newPath;
      if (this.currentFile === oldPath) {
        this.currentFile = newPath;
      }
    }

    this.updateTemplateExplorer();
    this.renderTabs();
    this.logInfo(`Moved file: ${oldPath} → ${newPath}`);
  }

  moveFolder(oldPath, newPath) {
    // Get all files in the folder
    const filesToMove = [];
    for (const [filePath] of this.templateFS.files) {
      if (filePath.startsWith(oldPath + '/') || filePath === oldPath) {
        filesToMove.push(filePath);
      }
    }

    // Move all files
    for (const filePath of filesToMove) {
      const relativePath = filePath.substring(oldPath.length);
      const newFilePath = newPath + relativePath;
      const content = this.templateFS.getFile(filePath);
      this.templateFS.setFile(newFilePath, content);

      // Update tabs if file is open
      const tabIndex = this.openTabs.indexOf(filePath);
      if (tabIndex !== -1) {
        this.openTabs[tabIndex] = newFilePath;
        if (this.currentFile === filePath) {
          this.currentFile = newFilePath;
        }
      }
    }

    // Delete old folder
    this.templateFS.deleteFolder(oldPath);

    // Add new folder explicitly
    this.templateFS.folders.add(newPath);

    this.updateTemplateExplorer();
    this.renderTabs();
    this.logInfo(`Moved folder: ${oldPath} → ${newPath} (${filesToMove.length} items)`);
  }
}

// Initialize app when DOM is ready
window.addEventListener('DOMContentLoaded', () => {
  window.app = new PlaygroundApp();
});
