import * as vscode from 'vscode';
import { getFlipClient } from '../flip-client';

/**
 * Show extension settings/configuration
 */
export async function showSettings(): Promise<void> {
  const options = await vscode.window.showQuickPick(
    [
      { label: '$(gear) Executable Path', description: 'Path to flip CLI', value: 'executablePath' },
      { label: '$(notification) Show Status Bar', description: 'Display brain status in status bar', value: 'showStatusBar' },
      { label: '$(info) Icon Theme', description: 'Icon style: mono, color, or mixed', value: 'iconTheme' },
      { label: '$(book) Default Journal Type', description: 'Format: daily, weekly, or combined', value: 'defaultJournalType' },
      { label: '$(file) Auto-create Tasks File', description: 'Create tasks.md automatically', value: 'autoCreateTasksFile' },
    ],
    { placeHolder: 'What would you like to configure?' }
  );

  if (!options) {
    return;
  }

  const config = vscode.workspace.getConfiguration('flip');

  switch (options.value) {
    case 'executablePath':
      await editExecutablePath(config);
      break;
    case 'showStatusBar':
      await toggleStatusBar(config);
      break;
    case 'iconTheme':
      await selectIconTheme(config);
      break;
    case 'defaultJournalType':
      await selectJournalType(config);
      break;
    case 'autoCreateTasksFile':
      await toggleAutoCreateTasks(config);
      break;
  }
}

/**
 * Edit the flip executable path
 */
async function editExecutablePath(config: vscode.WorkspaceConfiguration): Promise<void> {
  const current = config.get('executablePath', 'flip');
  
  const input = await vscode.window.showInputBox({
    prompt: 'Path to flip executable',
    value: current,
    placeHolder: 'flip',
  });

  if (input === undefined) {
    return;
  }

  try {
    await config.update('executablePath', input || 'flip', vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Executable path set to: ${input || 'flip'}`);
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to update executable path: ${error}`);
  }
}

/**
 * Toggle status bar visibility
 */
async function toggleStatusBar(config: vscode.WorkspaceConfiguration): Promise<void> {
  const current = config.get('showStatusBar', true);
  const newValue = !current;

  try {
    await config.update('showStatusBar', newValue, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Status bar visibility: ${newValue ? 'ON' : 'OFF'}`);
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to update status bar setting: ${error}`);
  }
}

/**
 * Select icon theme
 */
async function selectIconTheme(config: vscode.WorkspaceConfiguration): Promise<void> {
  const current = config.get('iconTheme', 'mixed');
  
  const selected = await vscode.window.showQuickPick(
    [
      { label: 'Mixed (recommended)', value: 'mixed', description: 'Icons + Text' },
      { label: 'Color', value: 'color', description: 'Colored icons' },
      { label: 'Monochrome', value: 'mono', description: 'Plain icons' },
    ],
    { placeHolder: 'Select icon theme', canPickMany: false }
  );

  if (!selected) {
    return;
  }

  try {
    await config.update('iconTheme', selected.value, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Icon theme set to: ${selected.label}`);
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to update icon theme: ${error}`);
  }
}

/**
 * Select default journal type
 */
async function selectJournalType(config: vscode.WorkspaceConfiguration): Promise<void> {
  const current = config.get('defaultJournalType', 'daily');
  
  const selected = await vscode.window.showQuickPick(
    [
      { label: 'Daily', value: 'daily', description: 'New entry every day' },
      { label: 'Weekly', value: 'weekly', description: 'New entry every week' },
      { label: 'Combined', value: 'combined', description: 'Single journal file' },
    ],
    { placeHolder: 'Select default journal type', canPickMany: false }
  );

  if (!selected) {
    return;
  }

  try {
    await config.update('defaultJournalType', selected.value, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Journal type set to: ${selected.label}`);
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to update journal type: ${error}`);
  }
}

/**
 * Toggle auto-create tasks file
 */
async function toggleAutoCreateTasks(config: vscode.WorkspaceConfiguration): Promise<void> {
  const current = config.get('autoCreateTasksFile', false);
  const newValue = !current;

  try {
    await config.update('autoCreateTasksFile', newValue, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Auto-create tasks file: ${newValue ? 'ON' : 'OFF'}`);
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to update auto-create setting: ${error}`);
  }
}

/**
 * Show manage menu (resources, templates, etc.)
 */
export async function showManage(): Promise<void> {
  const actions = await vscode.window.showQuickPick(
    [
      { label: '$(settings-gear) Settings', value: 'settings', description: 'Configure extension' },
      { label: '$(trash) Clean Up', value: 'cleanup', description: 'Remove temporary files' },
      { label: '$(refresh) Refresh Cache', value: 'cache', description: 'Clear cached data' },
      { label: '$(info) Diagnostics', value: 'diag', description: 'Show system information' },
    ],
    { placeHolder: 'What would you like to manage?' }
  );

  if (!actions) {
    return;
  }

  switch (actions.value) {
    case 'settings':
      await showSettings();
      break;
    case 'cleanup':
      await cleanupTemporaryFiles();
      break;
    case 'cache':
      await refreshCache();
      break;
    case 'diag':
      await showDiagnostics();
      break;
  }
}

/**
 * Clean up temporary/cache files
 */
async function cleanupTemporaryFiles(): Promise<void> {
  try {
    const client = getFlipClient();
    
    // Run flip command to cleanup
    const result = await client.runCommand(['workspace', 'cleanup']);

    if (result.success) {
      vscode.window.showInformationMessage('Cleanup completed successfully');
    } else {
      vscode.window.showErrorMessage(`Cleanup failed: ${result.error}`);
    }
  } catch (error) {
    vscode.window.showErrorMessage(`Error during cleanup: ${error}`);
  }
}

/**
 * Refresh/clear cached data
 */
async function refreshCache(): Promise<void> {
  try {
    const { refreshFlipClient } = await import('../flip-client');
    
    refreshFlipClient();
    vscode.window.showInformationMessage('Cache refreshed successfully');
  } catch (error) {
    vscode.window.showErrorMessage(`Error refreshing cache: ${error}`);
  }
}

/**
 * Show diagnostics/system information
 */
async function showDiagnostics(): Promise<void> {
  const client = getFlipClient();
  
  const info = await client.getInfo();
  const available = await client.isAvailable();

  if (!info.success || !info.data) {
    vscode.window.showErrorMessage('Could not retrieve system information');
    return;
  }

  const diagnosticsMarkdown = `
# Flip Diagnostics

**Status:** ${available ? '✅ Running' : '❌ Not available'}

**Active Brain:** \`${info.data.active_brain || 'None'}\`

**Brains:** ${info.data.brains?.length || 0}

## Extension Information
- **Version:** ${vscode.extensions.getExtension('danorama.flip-vscode')?.packageJSON.version || 'Unknown'}
- **VS Code Version:** ${vscode.version}

## System
- **Platform:** ${process.platform}
- **Architecture:** ${process.arch}
- **Node Version:** ${process.version}

## Configuration
- **Icon Theme:** ${vscode.workspace.getConfiguration('flip').get('iconTheme', 'mixed')}
- **Show Status Bar:** ${vscode.workspace.getConfiguration('flip').get('showStatusBar', true)}
`;

  // Create a new text document
  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: diagnosticsMarkdown,
  });

  await vscode.window.showTextDocument(doc, { preview: true });
}
