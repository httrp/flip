import * as vscode from 'vscode';
import { getFlipClient, FlipResult, JournalResult } from '../flip-client';

/**
 * Create or open today's journal
 */
export async function createJournal(): Promise<void> {
  const client = getFlipClient();

  // Show progress
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating journal...',
      cancellable: false,
    },
    async () => {
      const result = await client.createJournal();

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create journal: ${result.error}`);
        return;
      }

      const data = result.data as JournalResult;

      // Open the file
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      // Show notification
      const action = data.action === 'created' ? 'Created' : 'Opened';
      vscode.window.showInformationMessage(
        `${action} journal for ${data.date} in ${data.brain_name}`
      );
    }
  );
}
