import * as path from 'path';
import * as vscode from 'vscode';
import { getFlipClient, TaskResult } from '../flip-client';

/**
 * Location options for task creation
 */
interface TaskLocation {
  label: string;
  description?: string;
  value: 'default' | 'cursor' | 'custom';
}

/**
 * Create a new task with VS Code input dialogs
 */
export async function createTask(): Promise<void> {
  const client = getFlipClient();
  const editor = vscode.window.activeTextEditor;

  // Get description
  const description = await vscode.window.showInputBox({
    prompt: 'Task description',
    placeHolder: 'What needs to be done?',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Description is required';
      }
      return null;
    },
  });

  if (!description) {
    return; // User cancelled
  }

  // Get location option (where to insert the task)
  const locationOptions: TaskLocation[] = [
    { label: '$(file) Default task file', description: 'tasks/todo.md or similar', value: 'default' },
  ];

  // Add cursor option if we have an active markdown editor
  if (editor && editor.document.languageId === 'markdown') {
    const relPath = vscode.workspace.asRelativePath(editor.document.uri);
    locationOptions.push({
      label: '$(edit) At cursor position',
      description: relPath,
      value: 'cursor',
    });
  }

  locationOptions.push({
    label: '$(folder-opened) Choose file...',
    value: 'custom',
  });

  const selectedLocation = await vscode.window.showQuickPick(locationOptions, {
    placeHolder: 'Where to create the task?',
  });

  if (!selectedLocation) {
    return; // User cancelled
  }

  let file: string | undefined;
  let line: number | undefined;

  if (selectedLocation.value === 'cursor' && editor) {
    file = editor.document.uri.fsPath;
    line = editor.selection.active.line + 1; // Convert to 1-based
  } else if (selectedLocation.value === 'custom') {
    // Let user pick a markdown file
    const uris = await vscode.window.showOpenDialog({
      canSelectMany: false,
      filters: { 'Markdown': ['md'] },
      title: 'Select file for task',
    });
    if (uris && uris.length > 0) {
      file = uris[0].fsPath;
    } else {
      return; // User cancelled
    }
  }

  // Get due date (optional)
  const dueOptions = [
    { label: 'No due date', value: '' },
    { label: '$(calendar) Today', value: 'today' },
    { label: '$(arrow-right) Tomorrow', value: 'tomorrow' },
    { label: '$(edit) Custom date...', value: 'custom' },
  ];

  const selectedDue = await vscode.window.showQuickPick(dueOptions, {
    placeHolder: 'Due date (optional)',
  });

  let due: string | undefined;
  if (selectedDue?.value === 'custom') {
    const customDate = await vscode.window.showInputBox({
      prompt: 'Due date (YYYY-MM-DD)',
      placeHolder: '2025-01-15',
      validateInput: (value) => {
        if (value && !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
          return 'Please use YYYY-MM-DD format';
        }
        return null;
      },
    });
    due = customDate;
  } else if (selectedDue?.value) {
    due = selectedDue.value;
  }

  // Get priority (optional)
  const priorityOptions = [
    { label: '$(arrow-up) High', value: 'high' },
    { label: '$(dash) Medium', value: 'medium' },
    { label: '$(arrow-down) Low', value: 'low' },
    { label: 'No priority', value: '' },
  ];

  const selectedPriority = await vscode.window.showQuickPick(priorityOptions, {
    placeHolder: 'Priority (optional)',
  });

  const priority = selectedPriority?.value;

  // Ask if this is a frog task
  const frogOptions = [
    { label: 'No', value: false },
    { label: '$(target) 🐸 Yes - Eat the Frog!', description: 'Most important task to tackle first', value: true },
  ];

  const selectedFrog = await vscode.window.showQuickPick(frogOptions, {
    placeHolder: 'Is this your "eat the frog" task?',
  });

  const frog = selectedFrog?.value || false;

  // Get brain selection if multiple brains
  const info = await client.getInfo();
  let brain: string | undefined;

  if (info.success && info.data && info.data.brains.length > 1) {
    const brainItems = info.data.brains.map((b) => ({
      label: b.name,
      description: b.active ? '(active)' : '',
      detail: `${b.type} - ${b.path}`,
    }));

    const selected = await vscode.window.showQuickPick(brainItems, {
      placeHolder: 'Select brain for task',
    });

    if (selected) {
      brain = selected.label;
    }
  }

  // Save all open documents before creating task (to avoid merge conflicts with journal)
  await vscode.workspace.saveAll();

  // Create task
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating task...',
      cancellable: false,
    },
    async () => {
      const result = await client.createTask({
        description: description.trim(),
        brain,
        due,
        priority,
        frog,
        file,
        line,
      });

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create task: ${result.error}`);
        return;
      }

      const data = result.data as TaskResult;

      // If we created the task at cursor position in the currently open editor,
      // we need to refresh the editor since the CLI modified the file directly
      if (editor && editor.document.uri.fsPath === data.path && selectedLocation.value === 'cursor') {
        // Reload the document from disk to show CLI changes
        const fileUri = editor.document.uri;
        const savedVersion = editor.document.version;
        
        // Force reload by closing and reopening
        await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
        const doc = await vscode.workspace.openTextDocument(fileUri);
        const newEditor = await vscode.window.showTextDocument(doc);
        
        // Navigate to the line with the new task
        if (data.line) {
          const position = new vscode.Position(data.line - 1, 0);
          newEditor.selection = new vscode.Selection(position, position);
          newEditor.revealRange(new vscode.Range(position, position), vscode.TextEditorRevealType.InCenter);
        }
      } else {
        // Open the file normally
        const doc = await vscode.workspace.openTextDocument(data.path);
        await vscode.window.showTextDocument(doc);
      }

      let msg = `${frog ? '🐸 ' : ''}Created task in ${data.brain_name}`;
      if (data.due) {
        msg += ` (due: ${data.due})`;
      }
      vscode.window.showInformationMessage(`✓ ${msg} (linked in journal)`);
    }
  );
}

/**
 * Update task status at current cursor location
 */
export async function updateTaskStatus(): Promise<void> {
  const client = getFlipClient();
  const editor = vscode.window.activeTextEditor;

  if (!editor) {
    vscode.window.showWarningMessage('Open a task file to update its status.');
    return;
  }

  if (editor.document.languageId !== 'markdown') {
    vscode.window.showWarningMessage('Task status updates currently work in markdown files.');
    return;
  }

  const statusOptions: { label: string; value: 'open' | 'in-progress' | 'done' | 'deferred' | 'cancelled' }[] = [
    { label: '⬜ Open', value: 'open' },
    { label: '🔄 In Progress', value: 'in-progress' },
    { label: '✅ Done', value: 'done' },
    { label: '⏸️ Deferred', value: 'deferred' },
    { label: '❌ Cancelled', value: 'cancelled' },
  ];

  const selected = await vscode.window.showQuickPick(statusOptions, {
    placeHolder: 'Select new task status',
  });

  if (!selected) {
    return;
  }

  const file = editor.document.uri.fsPath;
  const line = editor.selection.active.line + 1;

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Updating task status...',
      cancellable: false,
    },
    async () => {
      const result = await client.updateTaskStatus({ file, line, status: selected.value });
      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Failed to update task: ${result.error ?? 'unknown error'}`);
        return;
      }

      const data = result.data as TaskResult;
      const doc = await vscode.workspace.openTextDocument(data.path);
      const editorInstance = await vscode.window.showTextDocument(doc);
      const position = new vscode.Position(data.line ? data.line - 1 : line - 1, 0);
      editorInstance.selection = new vscode.Selection(position, position);
      editorInstance.revealRange(new vscode.Range(position, position), vscode.TextEditorRevealType.InCenter);

      vscode.window.showInformationMessage(`${selected.label} in ${data.brain_name}`);

      // Add status change to today's journal
      await addTaskStatusChangeToJournal(data, selected.value);
    }
  );
}

/**
 * Quick toggle: Mark task as done (no prompt)
 * Perfect for keyboard shortcut or quick action
 */
export async function markTaskDone(): Promise<void> {
  const client = getFlipClient();
  const editor = vscode.window.activeTextEditor;

  if (!editor) {
    vscode.window.showWarningMessage('Open a task file to mark it as done.');
    return;
  }

  if (editor.document.languageId !== 'markdown') {
    vscode.window.showWarningMessage('Task operations work in markdown files.');
    return;
  }

  const file = editor.document.uri.fsPath;
  const line = editor.selection.active.line + 1;

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Marking task as done...',
      cancellable: false,
    },
    async () => {
      const result = await client.updateTaskStatus({ file, line, status: 'done' });
      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Failed to mark task as done: ${result.error ?? 'unknown error'}`);
        return;
      }

      const data = result.data as TaskResult;
      const doc = await vscode.workspace.openTextDocument(data.path);
      const editorInstance = await vscode.window.showTextDocument(doc);
      const position = new vscode.Position(data.line ? data.line - 1 : line - 1, 0);
      editorInstance.selection = new vscode.Selection(position, position);
      editorInstance.revealRange(new vscode.Range(position, position), vscode.TextEditorRevealType.InCenter);

      vscode.window.showInformationMessage(`✅ Task marked as done in ${data.brain_name}`);

      // Add status change to today's journal
      await addTaskStatusChangeToJournal(data, 'done');
    }
  );
}

/**
 * Inserts a task status-change entry into today's journal.
 * Silently ignores errors so it never blocks the main workflow.
 */
async function addTaskStatusChangeToJournal(data: TaskResult, newStatus: string): Promise<void> {
  const client = getFlipClient();

  try {
    // Ensure today's journal exists (creates if missing)
    const journalResult = await client.createJournal(data.brain_name);
    if (!journalResult.success || !journalResult.data) {
      return;
    }
    const journalUri = vscode.Uri.file(journalResult.data.path);
    const journalDocument = await vscode.workspace.openTextDocument(journalUri);

    const statusIcon: Record<string, string> = {
      'open': '⬜',
      'in-progress': '🔄',
      'done': '✅',
      'deferred': '⏸️',
      'cancelled': '❌',
    };
    const icon = statusIcon[newStatus] || '📋';

    const journalDir = path.dirname(journalDocument.uri.fsPath);
    const linkPath = path.relative(journalDir, data.path);
    const entry = `- ${icon} [${data.description}](${linkPath}) -> **${newStatus}**\n`;

    const content = journalDocument.getText();
    const tasksMatch = content.match(/^## (?:Tasks|Aufgaben)\s*$/m);
    const edit = new vscode.WorkspaceEdit();

    if (tasksMatch && tasksMatch.index !== undefined) {
      const headerLine = journalDocument.positionAt(tasksMatch.index).line;
      const insertPos = journalDocument.lineAt(headerLine).rangeIncludingLineBreak.end;
      edit.insert(journalUri, insertPos, entry);
    } else {
      const prefix = content.trimEnd().length > 0 ? '\n\n## Tasks\n' : '## Tasks\n';
      const insertPos = journalDocument.positionAt(content.length);
      edit.insert(journalUri, insertPos, `${prefix}${entry}`);
    }

    await vscode.workspace.applyEdit(edit);
    await journalDocument.save();
  } catch {
    // Silently ignore - journal update is best-effort
  }
}