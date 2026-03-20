import * as vscode from 'vscode';
import { getFlipClient, BrainSyncResult } from '../flip-client';

let syncOutput: vscode.OutputChannel | undefined;
function getSyncOutput(): vscode.OutputChannel {
  if (!syncOutput) { syncOutput = vscode.window.createOutputChannel('Flip Sync'); }
  return syncOutput;
}

/**
 * Sync all brains - commit changes with auto-generated messages
 */
export async function syncBrains(): Promise<void> {
  const client = getFlipClient();

  // Ask if user wants to push
  const pushChoice = await vscode.window.showQuickPick(
    [
      { label: '$(git-commit) Commit only', value: false, description: 'Commit changes locally' },
      { label: '$(cloud-upload) Commit & Push', value: true, description: 'Commit and push to remote' }
    ],
    {
      title: 'Flip: Sync Brains',
      placeHolder: 'Choose sync mode'
    }
  );

  if (!pushChoice) {
    return;
  }

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: pushChoice.value ? 'Syncing brains (commit & push)...' : 'Committing brain changes...',
      cancellable: false
    },
    async () => {
      const result = await client.sync({ push: pushChoice.value });

      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Sync failed: ${result.error || 'Unknown error'}`);
        return;
      }

      const brains = result.data.brains;
      
      // Count results
      const committed = brains.filter(b => b.committed);
      const noChanges = brains.filter(b => !b.has_changes && !b.error);
      const errors = brains.filter(b => b.error);

      // Show summary
      if (errors.length > 0) {
        const errorDetails = errors.map(b => `${b.name}: ${b.error}`).join('\n');
        vscode.window.showWarningMessage(
          `Sync completed with errors: ${committed.length} committed, ${errors.length} failed`,
          'Show Details'
        ).then(selection => {
          if (selection === 'Show Details') {
            showSyncDetails(brains);
          }
        });
      } else if (committed.length > 0) {
        const pushMsg = pushChoice.value ? ' and pushed' : '';
        vscode.window.showInformationMessage(
          `✅ ${committed.length} brain(s) committed${pushMsg}`,
          'Show Details'
        ).then(selection => {
          if (selection === 'Show Details') {
            showSyncDetails(brains);
          }
        });
      } else {
        vscode.window.showInformationMessage('No changes to commit in any brain');
      }
    }
  );
}

/**
 * Quick sync - commit all without prompts
 */
export async function quickSync(): Promise<void> {
  const client = getFlipClient();

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Quick syncing brains...',
      cancellable: false
    },
    async () => {
      const result = await client.sync({ push: false });

      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Sync failed: ${result.error || 'Unknown error'}`);
        return;
      }

      const brains = result.data.brains;
      const committed = brains.filter(b => b.committed);

      if (committed.length > 0) {
        vscode.window.showInformationMessage(`✅ ${committed.length} brain(s) committed`);
      } else {
        vscode.window.showInformationMessage('No changes to commit');
      }
    }
  );
}

/**
 * Sync and push all brains
 */
export async function syncAndPush(): Promise<void> {
  const client = getFlipClient();

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Syncing and pushing brains...',
      cancellable: false
    },
    async () => {
      const result = await client.sync({ push: true });

      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Sync failed: ${result.error || 'Unknown error'}`);
        return;
      }

      const brains = result.data.brains;
      const pushed = brains.filter(b => b.pushed);
      const committed = brains.filter(b => b.committed && !b.pushed);
      const errors = brains.filter(b => b.error);

      if (errors.length > 0) {
        vscode.window.showWarningMessage(
          `Sync completed with errors: ${pushed.length} pushed, ${errors.length} failed`,
          'Show Details'
        ).then(selection => {
          if (selection === 'Show Details') {
            showSyncDetails(brains);
          }
        });
      } else if (pushed.length > 0) {
        vscode.window.showInformationMessage(`✅ ${pushed.length} brain(s) synced and pushed`);
      } else if (committed.length > 0) {
        vscode.window.showInformationMessage(`✅ ${committed.length} brain(s) committed (no remote)`);
      } else {
        vscode.window.showInformationMessage('No changes to sync');
      }
    }
  );
}

/**
 * Show sync details in output channel
 */
function showSyncDetails(brains: BrainSyncResult[]): void {
  const output = getSyncOutput();
  output.clear();
  output.appendLine('=== Flip Sync Results ===\n');

  for (const brain of brains) {
    output.appendLine(`📁 ${brain.name}`);
    output.appendLine(`   Path: ${brain.path}`);
    
    if (!brain.has_changes && !brain.error) {
      output.appendLine('   Status: No changes');
    } else if (brain.error) {
      output.appendLine(`   Status: ❌ Error - ${brain.error}`);
    } else if (brain.committed) {
      output.appendLine('   Status: ✅ Committed');
      if (brain.pushed) {
        output.appendLine('           ✅ Pushed');
      }
      if (brain.commit_message) {
        output.appendLine(`   Message: ${brain.commit_message.split('\n')[0]}`);
      }
      if (brain.changed_files && brain.changed_files.length > 0) {
        output.appendLine('   Files:');
        for (const file of brain.changed_files) {
          output.appendLine(`     - ${file}`);
        }
      }
    }
    output.appendLine('');
  }

  output.show();
}
