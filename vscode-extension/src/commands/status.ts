import * as vscode from 'vscode';
import { getFlipClient } from '../flip-client';

/**
 * Show flip status as a markdown document in a new tab
 * This provides a comprehensive view of the current workspace state
 */
export async function showStatus(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getStatus();

  if (!result.success) {
    vscode.window.showErrorMessage(`Failed to get status: ${result.error}`);
    return;
  }

  const data = result.data!;
  
  // Build markdown content
  const markdown = buildStatusMarkdown(data);
  
  // Create a new untitled document with markdown content
  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: markdown
  });
  
  // Show the document in a new tab
  await vscode.window.showTextDocument(doc, {
    preview: true,
    viewColumn: vscode.ViewColumn.Beside
  });
}

/**
 * Build markdown content for status display
 */
function buildStatusMarkdown(data: any): string {
  const lines: string[] = [];
  
  // Header
  lines.push('# 🧠 Flip Status');
  lines.push('');
  lines.push(`**Workspace:** ${data.workspace.name}`);
  lines.push(`**Path:** \`${data.workspace.path}\``);
  lines.push('');
  
  // Active Brain
  if (data.active_brain) {
    const brain = data.active_brain;
    lines.push('## 📍 Active Brain');
    lines.push('');
    lines.push(`**Name:** ${brain.name}`);
    lines.push(`**Type:** ${brain.type}`);
    lines.push(`**Path:** \`${brain.path}\``);
    
    if (brain.git_branch) {
      lines.push('');
      lines.push('### 🔀 Git Status');
      lines.push(`**Branch:** ${brain.git_branch}`);
      lines.push(`**Status:** ${brain.git_status}`);
    }
    lines.push('');
  } else {
    lines.push('## ⚠️ No Active Brain');
    lines.push('');
  }
  
  // All Brains
  if (data.brains && data.brains.length > 0) {
    lines.push('## 🗂️ All Brains');
    lines.push('');
    
    for (const brain of data.brains) {
      const activeMarker = brain.active ? ' ✅' : '';
      lines.push(`### ${brain.name}${activeMarker}`);
      lines.push('');
      lines.push(`- **Type:** ${brain.type}`);
      lines.push(`- **Path:** \`${brain.path}\``);
      
      if (brain.git_branch) {
        lines.push(`- **Git:** ${brain.git_branch} (${brain.git_status})`);
      }
      
      lines.push('');
    }
  }
  
  // Stats if available
  if (data.stats) {
    lines.push('## 📊 Statistics');
    lines.push('');
    
    if (data.stats.notes !== undefined) {
      lines.push(`- **Notes:** ${data.stats.notes}`);
    }
    if (data.stats.tasks !== undefined) {
      lines.push(`- **Tasks:** ${data.stats.tasks}`);
    }
    if (data.stats.meetings !== undefined) {
      lines.push(`- **Meetings:** ${data.stats.meetings}`);
    }
    if (data.stats.journal_entries !== undefined) {
      lines.push(`- **Journal Entries:** ${data.stats.journal_entries}`);
    }
    
    lines.push('');
  }
  
  // Footer with timestamp
  const now = new Date().toLocaleString();
  lines.push('---');
  lines.push('');
  lines.push(`_Generated at ${now}_`);
  
  return lines.join('\n');
}

/**
 * Switch to a different brain
 */
export async function switchBrain(): Promise<void> {
  const client = getFlipClient();

  const info = await client.getInfo();

  if (!info.success) {
    vscode.window.showErrorMessage(`Failed to get brains: ${info.error}`);
    return;
  }

  if (!info.data || info.data.brains.length === 0) {
    vscode.window.showWarningMessage('No brains found');
    return;
  }

  const brainItems = info.data.brains.map((b) => ({
    label: b.name,
    description: b.active ? '✓ active' : '',
    detail: `${b.type} - ${b.path}`,
    brain: b,
  }));

  const selected = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Select brain to switch to',
  });

  if (!selected) {
    return;
  }

  // TODO: Implement brain switch when flip brain switch --json is available
  vscode.window.showInformationMessage(
    `Would switch to brain: ${selected.label} (not yet implemented)`
  );
}
