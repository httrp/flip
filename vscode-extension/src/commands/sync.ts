import * as vscode from 'vscode';
import { getFlipClient, BrainSyncResult } from '../flip-client';

let syncOutput: vscode.OutputChannel | undefined;
function getSyncOutput(): vscode.OutputChannel {
  if (!syncOutput) { syncOutput = vscode.window.createOutputChannel('Flip Sync'); }
  return syncOutput;
}

function isGitHubAuthError(error?: string): boolean {
  if (!error) {
    return false;
  }

  const lower = error.toLowerCase();
  return lower.includes('github authentication required for https remote')
    || lower.includes("could not read username for 'https://github.com'")
    || lower.includes('authentication failed');
}

function isGitHubSetupGitError(error?: string): boolean {
  if (!error) {
    return false;
  }

  return error.toLowerCase().includes('github cli is logged in, but git is not configured to use those credentials');
}

async function showSyncErrorSummary(brains: BrainSyncResult[], conflicts: BrainSyncResult[]): Promise<void> {
  const authErrors = brains.filter(brain => isGitHubAuthError(brain.error));
  const errors = brains.filter(brain => brain.error);

  if (authErrors.length > 0) {
    const setupGitErrors = authErrors.filter(brain => isGitHubSetupGitError(brain.error));
    const selection = await vscode.window.showWarningMessage(
      setupGitErrors.length > 0
        ? (setupGitErrors.length === 1
          ? 'Sync blocked: gh is logged in, but Git is not configured to use those credentials for one brain. Run gh auth setup-git or switch the remote to SSH.'
          : `Sync blocked: gh is logged in, but Git is not configured to use those credentials for ${setupGitErrors.length} brains. Run gh auth setup-git or switch the remotes to SSH.`)
        : (authErrors.length === 1
          ? 'Sync blocked: GitHub authorization is missing for one brain. Run gh auth login and gh auth setup-git, or switch the remote to SSH.'
          : `Sync blocked: GitHub authorization is missing for ${authErrors.length} brains. Run gh auth login and gh auth setup-git, or switch the remotes to SSH.`),
      'Show Details',
      'How To Fix'
    );

    if (selection === 'Show Details') {
      showSyncDetails(brains);
    } else if (selection === 'How To Fix') {
      showSyncAuthHelp(authErrors);
    }
    return;
  }

  if (errors.length > 0) {
    const selection = await vscode.window.showWarningMessage(
      conflicts.length > 0
        ? `Sync needs attention: ${conflicts.length} brain(s) have conflicts`
        : `Sync completed with errors: ${errors.length} brain(s) failed`,
      'Show Details'
    );

    if (selection === 'Show Details') {
      showSyncDetails(brains);
    }
  }
}

function showSyncAuthHelp(brains: BrainSyncResult[]): void {
  const output = getSyncOutput();
  output.clear();
  const setupGitOnly = brains.some(brain => isGitHubSetupGitError(brain.error));
  output.appendLine('=== Flip Sync Authorization Help ===');
  output.appendLine('');
  output.appendLine(setupGitOnly
    ? 'GitHub CLI is already logged in, but Git is not configured to use those credentials for HTTPS remotes.'
    : 'Flip uses Git non-interactively. For HTTPS remotes, Git must already have credentials available.');
  output.appendLine('');
  output.appendLine('Recommended setup:');
  if (!setupGitOnly) {
    output.appendLine('  1. gh auth login');
    output.appendLine('  2. gh auth setup-git');
  } else {
    output.appendLine('  1. gh auth setup-git');
  }
  output.appendLine('');
  output.appendLine('Alternative setup options:');
  output.appendLine('  - Configure a Git credential helper');
  output.appendLine('  - Switch the remote URL from HTTPS to SSH');
  output.appendLine('');
  output.appendLine('Affected brains:');
  for (const brain of brains) {
    output.appendLine(`  - ${brain.name}: ${brain.path}`);
  }
  output.appendLine('');
  output.appendLine('After setup, run Flip Sync again.');
  output.show();
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
        await showSyncErrorSummary(brains, conflicts);
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
      const errors = brains.filter(b => b.error);

      if (errors.length > 0) {
        await showSyncErrorSummary(brains, conflicts);
      } else if (conflicts.length > 0) {
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

  const authErrors = brains.filter(brain => isGitHubAuthError(brain.error));
  if (authErrors.length > 0) {
    output.appendLine('Authorization help:');
    output.appendLine('  Flip uses Git non-interactively. For HTTPS remotes, set up credentials first.');
    output.appendLine('  Recommended: gh auth login && gh auth setup-git');
    output.appendLine('  Alternatives: configure a Git credential helper or switch the remote to SSH.');
    output.appendLine('');
  }

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
