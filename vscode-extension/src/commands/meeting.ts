import * as vscode from 'vscode';
import { getFlipClient, NoteResult } from '../flip-client';

/**
 * Create a meeting note with minimal prompts
 */
export async function createMeetingNote(): Promise<void> {
  const client = getFlipClient();

  // Title
  const title = await vscode.window.showInputBox({
    prompt: 'Meeting title',
    placeHolder: 'Sprint Planning',
    validateInput: (v) => (!v || v.trim() === '' ? 'Title is required' : null),
  });
  if (!title) return;

  // Optional participants
  const participants = await vscode.window.showInputBox({
    prompt: 'Participants (comma-separated, optional)',
    placeHolder: 'Alice, Bob, Carol',
  });

  // Choose brain when multiple
  const info = await client.getInfo();
  let brain: string | undefined;
  if (info.success && info.data && info.data.brains.length > 1) {
    const pick = await vscode.window.showQuickPick(
      info.data.brains.map((b) => ({ label: b.name, description: b.active ? '(active)' : '' })),
      { placeHolder: 'Select brain' }
    );
    if (pick) brain = pick.label;
  }

  await vscode.window.withProgress(
    { location: vscode.ProgressLocation.Notification, title: 'Creating meeting note...', cancellable: false },
    async () => {
      const res = await client.createMeeting({ title: title.trim(), participants: participants?.trim(), brain });
      if (!res.success) {
        vscode.window.showErrorMessage(`Failed to create meeting note: ${res.error}`);
        return;
      }
      const data = res.data as NoteResult;
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      const choice = await vscode.window.showInformationMessage(
        `Created meeting note in ${data.brain_name}. Link in today's journal?`,
        'Yes',
        'No'
      );
      if (choice === 'Yes') {
        const linkRes = await client.addToJournal({ file: data.path, title: data.title, type: 'meeting', brain: data.brain_name });
        if (!linkRes.success) {
          vscode.window.showWarningMessage(`Failed to add to journal: ${linkRes.error}`);
        } else {
          vscode.window.showInformationMessage("✓ Linked in today's journal");
        }
      }
    }
  );
}
