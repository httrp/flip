import * as vscode from 'vscode';
import { getFlipClient, TaskInfo, TasksResult } from '../flip-client';

/**
 * Task Browser WebView Panel
 * Provides interactive task browsing with dual-axis filtering (scope + status)
 */
export class TaskBrowserPanel {
  public static readonly viewType = 'flip.taskBrowser';
  public static currentPanel: TaskBrowserPanel | undefined;

  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private _disposables: vscode.Disposable[] = [];
  private _allTasks: TaskInfo[] = [];
  private _filteredTasks: TaskInfo[] = [];
  private _scopeFilter: 'all' | 'my' = 'all';
  private _statusFilter: 'all' | 'open' | 'in-progress' | 'done' = 'all';

  public static createOrShow(extensionUri: vscode.Uri) {
    if (TaskBrowserPanel.currentPanel) {
      TaskBrowserPanel.currentPanel._panel.reveal(vscode.ViewColumn.One);
      return;
    }

    const panel = vscode.window.createWebviewPanel(
      TaskBrowserPanel.viewType,
      '📋 Task Browser',
      vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
        localResourceRoots: [],
      }
    );

    TaskBrowserPanel.currentPanel = new TaskBrowserPanel(panel, extensionUri);
  }

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri) {
    this._panel = panel;
    this._extensionUri = extensionUri;

    // Set initial HTML
    this._updateWebview();

    // Listen for when the panel is disposed
    this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

    // Handle messages from the webview
    this._panel.webview.onDidReceiveMessage(
      (message) => this._handleMessage(message),
      null,
      this._disposables
    );

    // Load tasks on init
    this._loadTasks();
  }

  private async _handleMessage(message: any) {
    switch (message.command) {
      case 'filter':
        this._scopeFilter = message.scope || 'all';
        this._statusFilter = message.status || 'all';
        this._applyFilters();
        this._updateWebview();
        break;
      case 'openTask':
        this._openTask(message.task);
        break;
      case 'refresh':
        this._loadTasks();
        break;
    }
  }

  private async _loadTasks() {
    try {
      this._panel.webview.postMessage({ command: 'loading' });

      const client = getFlipClient();
      const result = await client.browseTasksJson();

      if (result.success && result.data) {
        this._allTasks = result.data.tasks;
        this._applyFilters();
        this._updateWebview();
      } else {
        this._panel.webview.postMessage({
          command: 'error',
          message: 'Failed to load tasks',
        });
      }
    } catch (error) {
      this._panel.webview.postMessage({
        command: 'error',
        message: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
      });
    }
  }

  private _applyFilters() {
    let filtered = this._allTasks;

    // Apply status filter
    if (this._statusFilter !== 'all') {
      filtered = filtered.filter((t) => t.status === this._statusFilter);
    }

    // Apply scope filter (for future enhancement)
    if (this._scopeFilter === 'my') {
      // This could be enhanced to filter by organization or other criteria
    }

    this._filteredTasks = filtered;
  }

  private _updateWebview() {
    this._panel.webview.html = this._getHtmlContent();
  }

  private _getHtmlContent(): string {
    const tasks = this._filteredTasks;
    const tasksHtml = tasks
      .map(
        (task) => `
      <div class="task-item" data-path="${this._escapeHtml(task.path)}" data-line="${task.line}">
        <div class="task-header">
          <div class="task-icons">
            ${task.frog ? '<span class="icon frog" title="Frog task">🐸</span>' : ''}
            ${
              task.priority === 'high'
                ? '<span class="icon priority-high" title="High priority">⏫</span>'
                : task.priority === 'medium'
                  ? '<span class="icon priority-medium" title="Medium priority">🔼</span>'
                  : task.priority === 'low'
                    ? '<span class="icon priority-low" title="Low priority">🔽</span>'
                    : ''
            }
            ${task.status === 'in-progress' ? '<span class="icon in-progress" title="In progress">🔄</span>' : ''}
          </div>
          <div class="task-title">${this._escapeHtml(task.description)}</div>
          <div class="task-meta">
            ${task.due ? `<span class="due-date">📅 ${task.due}</span>` : ''}
            <span class="status" data-status="${task.status}">${task.status}</span>
          </div>
        </div>
        <div class="task-location">
          <span class="brain-name">${this._escapeHtml(task.brain_name)}</span>
          <span class="file-path">${this._escapeHtml(task.rel_path)}:${task.line}</span>
        </div>
        ${
          task.tags && task.tags.length > 0
            ? `<div class="task-tags">${task.tags.map((tag) => `<span class="tag">${this._escapeHtml(tag)}</span>`).join('')}</div>`
            : ''
        }
      </div>
    `
      )
      .join('');

    return `
      <!DOCTYPE html>
      <html>
      <head>
        <style>
          * {
            box-sizing: border-box;
          }
          body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            padding: 0;
            margin: 0;
            background-color: var(--vscode-editor-background);
            color: var(--vscode-editor-foreground);
          }
          .container {
            display: flex;
            flex-direction: column;
            height: 100vh;
          }
          .controls {
            display: flex;
            gap: 12px;
            padding: 12px 16px;
            background-color: var(--vscode-panel-background);
            border-bottom: 1px solid var(--vscode-panel-border);
            flex-shrink: 0;
          }
          .filter-group {
            display: flex;
            gap: 8px;
            align-items: center;
          }
          .filter-label {
            font-size: 12px;
            font-weight: 500;
            text-transform: uppercase;
            color: var(--vscode-descriptionForeground);
            min-width: 40px;
          }
          select {
            background-color: var(--vscode-input-background);
            color: var(--vscode-input-foreground);
            border: 1px solid var(--vscode-input-border);
            padding: 4px 8px;
            border-radius: 3px;
            font-size: 12px;
            cursor: pointer;
            min-width: 120px;
          }
          select:hover {
            border-color: var(--vscode-focusBorder);
          }
          button {
            background-color: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            padding: 6px 12px;
            border-radius: 3px;
            font-size: 12px;
            cursor: pointer;
            transition: background-color 0.2s;
          }
          button:hover {
            background-color: var(--vscode-button-hoverBackground);
          }
          .count {
            font-size: 12px;
            color: var(--vscode-descriptionForeground);
            margin-left: auto;
          }
          .tasks {
            flex: 1;
            overflow-y: auto;
            padding: 0;
          }
          .task-item {
            padding: 12px 16px;
            border-bottom: 1px solid var(--vscode-panel-border);
            cursor: pointer;
            transition: background-color 0.1s;
          }
          .task-item:hover {
            background-color: var(--vscode-list-hoverBackground);
          }
          .task-header {
            display: flex;
            gap: 12px;
            margin-bottom: 6px;
            align-items: flex-start;
          }
          .task-icons {
            display: flex;
            gap: 4px;
            flex-shrink: 0;
            min-width: 60px;
          }
          .icon {
            font-size: 14px;
            line-height: 1;
          }
          .task-title {
            flex: 1;
            font-weight: 500;
            font-size: 13px;
            word-break: break-word;
          }
          .task-meta {
            display: flex;
            gap: 8px;
            align-items: center;
            font-size: 12px;
          }
          .due-date {
            color: var(--vscode-descriptionForeground);
          }
          .status {
            padding: 2px 6px;
            border-radius: 2px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;
          }
          .status[data-status="open"] {
            background-color: rgba(244, 67, 54, 0.2);
            color: #f44336;
          }
          .status[data-status="in-progress"] {
            background-color: rgba(255, 152, 0, 0.2);
            color: #ff9800;
          }
          .status[data-status="done"] {
            background-color: rgba(76, 175, 80, 0.2);
            color: #4caf50;
          }
          .task-location {
            display: flex;
            gap: 8px;
            font-size: 11px;
            color: var(--vscode-descriptionForeground);
          }
          .brain-name {
            font-weight: 500;
            padding: 2px 6px;
            background-color: var(--vscode-input-background);
            border-radius: 2px;
          }
          .file-path {
            font-family: 'Menlo', 'Monaco', monospace;
          }
          .task-tags {
            display: flex;
            gap: 4px;
            flex-wrap: wrap;
            margin-top: 6px;
          }
          .tag {
            display: inline-block;
            padding: 2px 6px;
            background-color: var(--vscode-input-background);
            border-radius: 12px;
            font-size: 11px;
            color: var(--vscode-descriptionForeground);
          }
          .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
            color: var(--vscode-descriptionForeground);
          }
          .error {
            padding: 16px;
            background-color: rgba(244, 67, 54, 0.1);
            color: #f44336;
            border-radius: 3px;
            margin: 16px;
          }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="controls">
            <div class="filter-group">
              <label class="filter-label">Scope:</label>
              <select id="scopeFilter" onchange="handleFilterChange()">
                <option value="all">All Tasks</option>
                <option value="my">My Tasks</option>
              </select>
            </div>
            <div class="filter-group">
              <label class="filter-label">Status:</label>
              <select id="statusFilter" onchange="handleFilterChange()">
                <option value="all">All</option>
                <option value="open">Open</option>
                <option value="in-progress">In Progress</option>
                <option value="done">Done</option>
              </select>
            </div>
            <button onclick="refreshTasks()">🔄 Refresh</button>
            <div class="count">${tasks.length} tasks</div>
          </div>
          <div class="tasks" id="tasksList">
            ${tasksHtml}
          </div>
        </div>
        <script>
          const vscode = acquireVsCodeApi();

          function handleFilterChange() {
            const scope = document.getElementById('scopeFilter').value;
            const status = document.getElementById('statusFilter').value;
            vscode.postMessage({
              command: 'filter',
              scope,
              status,
            });
          }

          function refreshTasks() {
            vscode.postMessage({ command: 'refresh' });
          }

          // Handle task item clicks
          document.querySelectorAll('.task-item').forEach(item => {
            item.addEventListener('click', () => {
              const path = item.dataset.path;
              const line = parseInt(item.dataset.line, 10);
              vscode.postMessage({
                command: 'openTask',
                task: { path, line },
              });
            });
          });

          // Listen for messages from extension
          window.addEventListener('message', event => {
            const message = event.data;
            if (message.command === 'loading') {
              document.getElementById('tasksList').innerHTML = '<div class="loading">⏳ Loading tasks...</div>';
            } else if (message.command === 'error') {
              document.getElementById('tasksList').innerHTML = '<div class="error">❌ ' + message.message + '</div>';
            }
          });
        </script>
      </body>
      </html>
    `;
  }

  private _openTask(task: { path: string; line: number }) {
    vscode.workspace.openTextDocument(task.path).then((doc) => {
      vscode.window.showTextDocument(doc).then((editor) => {
        if (task.line > 0) {
          const line = task.line - 1;
          const range = new vscode.Range(line, 0, line, 0);
          editor.selection = new vscode.Selection(range.start, range.end);
          editor.revealRange(range, vscode.TextEditorRevealType.InCenter);
        }
      });
    });
  }

  public dispose() {
    TaskBrowserPanel.currentPanel = undefined;

    this._panel.dispose();

    while (this._disposables.length) {
      const x = this._disposables.pop();
      if (x) {
        x.dispose();
      }
    }
  }

  private _escapeHtml(text: string): string {
    const map: { [key: string]: string } = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#039;',
    };
    return text.replace(/[&<>"']/g, (m) => map[m]);
  }
}
