import * as vscode from 'vscode';
import { getFlipClient } from '../flip-client';

/**
 * Show flip status in an information message
 */
export async function showStatus(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getStatus();

  if (!result.success) {
    vscode.window.showErrorMessage(`Failed to get status: ${result.error}`);
    return;
  }

  const data = result.data!;
  const brainInfo = data.active_brain
    ? `Brain: ${data.active_brain.name} (${data.active_brain.type})`
    : 'No active brain';
  const gitInfo = data.active_brain?.git_branch
    ? ` | ${data.active_brain.git_branch} (${data.active_brain.git_status})`
    : '';

  vscode.window.showInformationMessage(
    `Flip: ${data.workspace.name} | ${brainInfo}${gitInfo}`
  );
}

/**
 * Switch to a different brain
 */
export async function switchBrain(): Promise<void> {
  const client = getFlipClient();

  const info = await client.getInfo();

  if (!info.success) {
    vscode.window.showErrorMessage(`Failed to get brains: ${info.error}`);
    return;
  }

  if (!info.data || info.data.brains.length === 0) {
    vscode.window.showWarningMessage('No brains found');
    return;
  }

  const brainItems = info.data.brains.map((b) => ({
    label: b.name,
    description: b.active ? '✓ active' : '',
    detail: `${b.type} - ${b.path}`,
    brain: b,
  }));

  const selected = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Select brain to switch to',
  });

  if (!selected) {
    return;
  }

  // TODO: Implement brain switch when flip brain switch --json is available
  vscode.window.showInformationMessage(
    `Would switch to brain: ${selected.label} (not yet implemented)`
  );
}
