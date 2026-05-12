import * as vscode from 'vscode';
import { getFlipClient, BrainSyncResult } from '../flip-client';

let syncOutput: vscode.OutputChannel | undefined;
function getSyncOutput(): vscode.OutputChannel {
  if (!syncOutput) { syncOutput = vscode.window.createOutputChannel('Flip Sync'); }
  return syncOutput;
}

/**
 * Sync all brains bidirectionally.
 */
export async function syncBrains(): Promise<void> {
  const client = getFlipClient();

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Syncing brains...',
      cancellable: false
    },
    async () => {
      const result = await client.sync();

      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Sync failed: ${result.error || 'Unknown error'}`);
        return;
      }

      const brains = result.data.brains;
      const synced = brains.filter(b => b.pulled || b.committed || b.pushed);
      const conflicts = brains.filter(b => b.has_conflicts);
      const errors = brains.filter(b => b.error);

      if (errors.length > 0) {
        vscode.window.showWarningMessage(
          conflicts.length > 0
            ? `Sync needs attention: ${conflicts.length} brain(s) have conflicts`
            : `Sync completed with errors: ${errors.length} brain(s) failed`,
          'Show Details'
        ).then(selection => {
          if (selection === 'Show Details') {
            showSyncDetails(brains);
          }
        });
      } else if (synced.length > 0) {
        vscode.window.showInformationMessage(
          `✅ ${synced.length} brain(s) synced`,
          'Show Details'
        ).then(selection => {
          if (selection === 'Show Details') {
            showSyncDetails(brains);
          }
        });
      } else {
        vscode.window.showInformationMessage('Brains are already up to date');
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
      const result = await client.sync();

      if (!result.success || !result.data) {
        vscode.window.showErrorMessage(`Sync failed: ${result.error || 'Unknown error'}`);
        return;
      }

      const brains = result.data.brains;
      const synced = brains.filter(b => b.pulled || b.committed || b.pushed);
      const conflicts = brains.filter(b => b.has_conflicts);

      if (conflicts.length > 0) {
        vscode.window.showWarningMessage(`Sync needs attention: ${conflicts.length} brain(s) have conflicts`, 'Show Details')
          .then(selection => {
            if (selection === 'Show Details') {
              showSyncDetails(brains);
            }
          });
      } else if (synced.length > 0) {
        vscode.window.showInformationMessage(`✅ ${synced.length} brain(s) synced`);
      } else {
        vscode.window.showInformationMessage('Brains are already up to date');
      }
    }
  );
}

/**
 * Sync and push all brains
 */
export async function syncAndPush(): Promise<void> {
  await syncBrains();
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

    if (brain.error) {
      output.appendLine(`   Status: ❌ Error - ${brain.error}`);
      if (brain.conflict_files && brain.conflict_files.length > 0) {
        output.appendLine('   Conflict files:');
        for (const file of brain.conflict_files) {
          output.appendLine(`     - ${file}`);
        }
      }
    } else if (!brain.pulled && !brain.committed && !brain.pushed) {
      output.appendLine('   Status: Up to date');
    } else {
      output.appendLine('   Status: ✅ Synced');
      if (brain.pulled) {
        output.appendLine('           ✅ Pulled');
      }
      if (brain.committed) {
        output.appendLine('           ✅ Committed');
      }
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
