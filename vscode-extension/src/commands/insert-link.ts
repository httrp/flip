import * as vscode from 'vscode';
import { getFlipClient, NoteInfo } from '../flip-client';

/**
 * Insert a link to another note in the current document
 */
export async function insertLink(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage('No active editor');
    return;
  }

  const client = getFlipClient();

  // Get current file path to determine brain
  const currentFile = editor.document.uri.fsPath;
  
  // Get brain info to find which brain we're in
  const info = await client.getInfo();
  if (!info.success || !info.data) {
    vscode.window.showErrorMessage('Failed to get brain info');
    return;
  }

  // Find which brain the current file belongs to
  let currentBrain = info.data.brains.find(b => currentFile.startsWith(b.path));
  
  // If not in a brain, use active brain
  if (!currentBrain) {
    currentBrain = info.data.brains.find(b => b.active);
  }

  if (!currentBrain) {
    vscode.window.showErrorMessage('Could not determine current brain');
    return;
  }

  // Get notes for this brain
  const notesResult = await client.getNotes(currentBrain.name);
  if (!notesResult.success || !notesResult.data) {
    vscode.window.showErrorMessage('Failed to get notes list');
    return;
  }

  const notes = notesResult.data.notes;
  if (notes.length === 0) {
    vscode.window.showInformationMessage('No notes found in this brain');
    return;
  }

  // Create QuickPick items
  const items: (vscode.QuickPickItem & { note: NoteInfo })[] = notes.map(note => ({
    label: note.name,
    description: note.rel_path,
    detail: `Insert: ${note.link_format}`,
    note: note
  }));

  // Show QuickPick with search
  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: `Search notes in ${currentBrain.name} (${notesResult.data.link_syntax})`,
    matchOnDescription: true,
    matchOnDetail: false
  });

  if (!selected) {
    return; // User cancelled
  }

  // Insert the link at cursor position
  const linkText = selected.note.link_format;
  
  await editor.edit(editBuilder => {
    const selection = editor.selection;
    if (selection.isEmpty) {
      // Insert at cursor
      editBuilder.insert(selection.active, linkText);
    } else {
      // Replace selection
      editBuilder.replace(selection, linkText);
    }
  });
}
