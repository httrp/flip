import * as vscode from 'vscode';
import { getFlipClient, NoteResult } from '../flip-client';
import { promptAutoAddToJournal } from './journal-helper';

/**
 * Create a new note with VS Code input dialogs
 */
export async function createNote(): Promise<void> {
  const client = getFlipClient();

  // Get title
  const title = await vscode.window.showInputBox({
    prompt: 'Note title',
    placeHolder: 'My new note',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Title is required';
      }
      return null;
    },
  });

  if (!title) {
    return; // User cancelled
  }

  // Get tags (optional)
  const tags = await vscode.window.showInputBox({
    prompt: 'Tags (comma-separated, optional)',
    placeHolder: 'work, project, idea',
  });

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
      placeHolder: 'Select brain',
    });

    if (selected) {
      brain = selected.label;
    }
  }

  // Create note
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating note...',
      cancellable: false,
    },
    async () => {
      const result = await client.createNote({
        title: title.trim(),
        tags: tags?.trim(),
        brain,
      });

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create note: ${result.error}`);
        return;
      }

      const data = result.data as NoteResult;

      // Open the file
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      // Auto-add to journal with countdown (default = add)
      await promptAutoAddToJournal(data, 'note');
    }
  );
}

/**
 * Create a quick note (minimal prompts - just title)
 */
export async function createQuicknote(): Promise<void> {
  const client = getFlipClient();

  // Get title
  const title = await vscode.window.showInputBox({
    prompt: 'Quick note title',
    placeHolder: 'Quick thought...',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Title is required';
      }
      return null;
    },
  });

  if (!title) {
    return; // User cancelled
  }

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating quick note...',
      cancellable: false,
    },
    async () => {
      const result = await client.createQuicknote({ title: title.trim() });

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create quick note: ${result.error}`);
        return;
      }

      const data = result.data as NoteResult;

      // Open the file
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      const choice = await vscode.window.showInformationMessage(
        `Created quick note in ${data.brain_name}. Link in today's journal?`,
        'Yes',
        'No'
      );
      if (choice === 'Yes') {
        const linkRes = await client.addToJournal({ file: data.path, type: 'note', brain: data.brain_name });
        if (!linkRes.success) {
          vscode.window.showWarningMessage(`Failed to add to journal: ${linkRes.error}`);
        } else {
          vscode.window.showInformationMessage('✓ Linked in today\'s journal');
        }
      }
    }
  );
}
