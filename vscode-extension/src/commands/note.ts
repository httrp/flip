import * as vscode from 'vscode';
import { getFlipClient, NoteResult } from '../flip-client';

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

      vscode.window.showInformationMessage(
        `Created note "${data.title}" in ${data.brain_name}`
      );
    }
  );
}

/**
 * Create a quick note (minimal prompts)
 */
export async function createQuicknote(): Promise<void> {
  const client = getFlipClient();

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating quick note...',
      cancellable: false,
    },
    async () => {
      const result = await client.createQuicknote();

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create quick note: ${result.error}`);
        return;
      }

      const data = result.data as NoteResult;

      // Open the file
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      vscode.window.showInformationMessage(`Created quick note in ${data.brain_name}`);
    }
  );
}
