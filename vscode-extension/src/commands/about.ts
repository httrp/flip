import * as vscode from 'vscode';
import { getFlipClient } from '../flip-client';

/**
 * Show About information with version, links, and documentation
 */
export async function showAbout(): Promise<void> {
  const client = getFlipClient();
  
  // Get flip version
  const info = await client.getInfo();
  const flipVersion = info.success && info.data ? info.data.flip_version : 'unknown';
  
  // Get extension version from package.json
  const extension = vscode.extensions.getExtension('danorama.flip-vscode');
  const extensionVersion = extension?.packageJSON.version || 'unknown';
  
  // Build markdown content
  const markdown = buildAboutMarkdown(flipVersion, extensionVersion);
  
  // Create a new untitled document with markdown content
  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: markdown
  });
  
  // Show the document in a new tab
  await vscode.window.showTextDocument(doc, {
    preview: true,
    viewColumn: vscode.ViewColumn.Active
  });
}

/**
 * Show Help/Documentation
 */
export async function showHelp(): Promise<void> {
  // Try to find README in flip repo
  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  
  if (!workspaceFolder) {
    vscode.window.showWarningMessage('No workspace folder open');
    return;
  }
  
  // Build help markdown
  const markdown = buildHelpMarkdown();
  
  // Create a new untitled document with markdown content
  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: markdown
  });
  
  // Show the document in a new tab
  await vscode.window.showTextDocument(doc, {
    preview: true,
    viewColumn: vscode.ViewColumn.Active
  });
}

/**
 * Build About markdown content
 */
function buildAboutMarkdown(flipVersion: string, extensionVersion: string): string {
  const lines: string[] = [];
  
  lines.push('# 🧠 Flip - Personal Knowledge Management');
  lines.push('');
  lines.push('> Your intelligent 2nd brain assistant');
  lines.push('');
  
  lines.push('## 📦 Version Information');
  lines.push('');
  lines.push(`- **Flip CLI:** ${flipVersion}`);
  lines.push(`- **VS Code Extension:** ${extensionVersion}`);
  lines.push('');
  
  lines.push('## 🔗 Links');
  lines.push('');
  lines.push('- [GitHub Repository](https://github.com/httrp/flip)');
  lines.push('- [Documentation](https://github.com/httrp/flip/blob/main/README.md)');
  lines.push('- [Report Issue](https://github.com/httrp/flip/issues)');
  lines.push('');
  
  lines.push('## ✨ Features');
  lines.push('');
  lines.push('- 📝 **Notes & Journals** - Create and organize your thoughts');
  lines.push('- ✅ **Tasks** - Manage your todos with priorities and frogs');
  lines.push('- 🤝 **Meetings** - Document meetings with participants');
  lines.push('- 🔍 **Search** - Find anything across all brains');
  lines.push('- 🔄 **Sync** - Git-based synchronization');
  lines.push('- 🧩 **Multi-Brain** - Organize knowledge in separate brains');
  lines.push('- 📋 **Templates** - Reusable document templates');
  lines.push('- 💪 **Exercises** - Track your fitness activities');
  lines.push('');
  
  lines.push('## 🎯 Supported Brain Types');
  lines.push('');
  lines.push('- Flip (native format)');
  lines.push('- Obsidian');
  lines.push('- Logseq');
  lines.push('- Dendron');
  lines.push('- Foam');
  lines.push('');
  
  lines.push('---');
  lines.push('');
  lines.push('_Made with ❤️ by danorama_');
  
  return lines.join('\n');
}

/**
 * Build Help markdown content
 */
function buildHelpMarkdown(): string {
  const lines: string[] = [];
  
  lines.push('# 📚 Flip Help & Quick Reference');
  lines.push('');
  
  lines.push('## 🚀 Getting Started');
  lines.push('');
  lines.push('1. **Create Journal:** `Cmd+Shift+P` → "Flip: Create Journal"');
  lines.push('2. **Create Note:** `Cmd+Shift+P` → "Flip: Create Note"');
  lines.push('3. **Create Task:** `Cmd+Shift+P` → "Flip: Create Task"');
  lines.push('');
  
  lines.push('## 📝 Common Commands');
  lines.push('');
  lines.push('### Creating Content');
  lines.push('- **Flip: Create Journal** - Create or open today\'s journal');
  lines.push('- **Flip: Create Note** - Create a new note with tags');
  lines.push('- **Flip: Quick Note** - Fast note creation');
  lines.push('- **Flip: Create Task** - Create a new task');
  lines.push('- **Flip: Create Meeting** - Document a meeting (via CLI)');
  lines.push('');
  
  lines.push('### Tasks');
  lines.push('- **Flip: Task Search** - Search and filter tasks');
  lines.push('- **Flip: Show Frogs** - View your most important tasks (🐸)');
  lines.push('- **Flip: Today\'s Tasks** - Tasks due today');
  lines.push('- **Flip: Update Task Status** - Change task status');
  lines.push('');
  
  lines.push('### Search & Browse');
  lines.push('- **Flip: Search** - Search across all brains');
  lines.push('- **Flip: Recent Notes** - Browse recently modified notes');
  lines.push('- **Flip: Search Filtered** - Advanced search with filters');
  lines.push('');
  
  lines.push('### Organization');
  lines.push('- **Flip: Switch Brain** - Switch between brains');
  lines.push('- **Flip: Insert Link** - Insert a link to another file');
  lines.push('- **Flip: Add to Journal** - Link current file to journal');
  lines.push('- **Flip: Journal Sync** - Check for missing journal entries');
  lines.push('');
  
  lines.push('### Templates & Definitions');
  lines.push('- **Flip: Templates** - Browse and use templates');
  lines.push('- **Flip: Definitions** - Manage people/organizations');
  lines.push('- **Flip: Add Person** - Add a person to definitions');
  lines.push('- **Flip: Add Organization** - Add an organization');
  lines.push('');
  
  lines.push('### Sync & Maintenance');
  lines.push('- **Flip: Sync** - Sync all brains with git');
  lines.push('- **Flip: Quick Sync** - Fast commit + sync');
  lines.push('- **Flip: Brain Health** - Check brain for issues');
  lines.push('- **Flip: Status** - View workspace status');
  lines.push('');
  
  lines.push('## 🐸 What are Frogs?');
  lines.push('');
  lines.push('Frogs are your **most important tasks** - the ones you should "eat first".');
  lines.push('Mark tasks as frogs with `frog: true` in the frontmatter.');
  lines.push('');
  
  lines.push('## 🏷️ Task Priorities');
  lines.push('');
  lines.push('- `⏫ high` - High priority');
  lines.push('- `🔼 medium` - Medium priority (default)');
  lines.push('- `🔽 low` - Low priority');
  lines.push('');
  
  lines.push('## 📋 Task Statuses');
  lines.push('');
  lines.push('- `open` - Not started');
  lines.push('- `in-progress` - Currently working on');
  lines.push('- `done` - Completed');
  lines.push('- `cancelled` - Cancelled/dropped');
  lines.push('');
  
  lines.push('## 💡 Tips');
  lines.push('');
  lines.push('- Use `Cmd+K Cmd+I` to insert links between notes');
  lines.push('- Journal entries auto-link to notes/tasks created that day');
  lines.push('- Templates can include placeholders like `{{date}}`, `{{title}}`');
  lines.push('- Brain Health checks for common issues like broken links');
  lines.push('');
  
  lines.push('---');
  lines.push('');
  lines.push('For more information, visit the [GitHub Repository](https://github.com/httrp/flip)');
  
  return lines.join('\n');
}
