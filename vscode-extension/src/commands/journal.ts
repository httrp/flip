import * as vscode from 'vscode';
import { getFlipClient, FlipResult, JournalResult, BrainInfo } from '../flip-client';

/**
 * Create or open today's journal
 */
export async function createJournal(): Promise<void> {
  const client = getFlipClient();

  // Get available brains
  const info = await client.getInfo();
  if (!info.success || !info.data) {
    vscode.window.showErrorMessage('Failed to get brain info');
    return;
  }

  // If multiple brains, let user choose
  let selectedBrain: BrainInfo | undefined;
  
  if (info.data.brains.length > 1) {
    const brainLabels = info.data.brains.map(b => 
      b.active ? `${b.name} (active)` : b.name
    );
    const selectedIndex = await vscode.window.showQuickPick(brainLabels, {
      placeHolder: 'Select a brain for the journal',
    });
    
    if (selectedIndex === undefined) {
      return; // User cancelled
    }
    
    const idx = brainLabels.indexOf(selectedIndex);
    selectedBrain = info.data.brains[idx];
  } else if (info.data.brains.length === 1) {
    selectedBrain = info.data.brains[0];
  } else {
    vscode.window.showErrorMessage('No brains found');
    return;
  }

  // Show progress
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating journal...',
      cancellable: false,
    },
    async () => {
      const result = await client.createJournal(selectedBrain?.name);

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
