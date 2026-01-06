import * as vscode from 'vscode';
import { getFlipClient, refreshFlipClient, BrainInfo } from './flip-client';
import { createJournal } from './commands/journal';
import { createNote, createQuicknote } from './commands/note';
import { createTask } from './commands/task';
import { showStatus, switchBrain } from './commands/status';
import { insertLink } from './commands/insert-link';

let statusBarItem: vscode.StatusBarItem | undefined;

export function activate(context: vscode.ExtensionContext) {
  console.log('Flip extension activated');

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand('flip.journal', createJournal),
    vscode.commands.registerCommand('flip.note', createNote),
    vscode.commands.registerCommand('flip.quicknote', createQuicknote),
    vscode.commands.registerCommand('flip.taskNew', createTask),
    vscode.commands.registerCommand('flip.status', showStatus),
    vscode.commands.registerCommand('flip.switchBrain', switchBrain),
    vscode.commands.registerCommand('flip.insertLink', insertLink),
    
    // Placeholder commands (to be implemented)
    vscode.commands.registerCommand('flip.meetingNote', () => {
      vscode.window.showInformationMessage('Meeting note: Not yet implemented');
    }),
    vscode.commands.registerCommand('flip.search', () => {
      vscode.window.showInformationMessage('Search: Not yet implemented');
    }),
    vscode.commands.registerCommand('flip.taskDone', () => {
      vscode.window.showInformationMessage('Task done: Not yet implemented');
    })
  );

  // Create status bar item
  const config = vscode.workspace.getConfiguration('flip');
  if (config.get('showStatusBar', true)) {
    statusBarItem = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );
    statusBarItem.command = 'flip.switchBrain';
    statusBarItem.tooltip = 'Click to switch brain';
    context.subscriptions.push(statusBarItem);

    // Update status bar
    updateStatusBar();
  }

  // Listen for config changes
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('flip')) {
        refreshFlipClient();
        updateStatusBar();
      }
    })
  );

  // Check if flip is available
  checkFlipAvailable();
}

async function checkFlipAvailable(): Promise<void> {
  const client = getFlipClient();
  const available = await client.isAvailable();

  if (!available) {
    vscode.window.showWarningMessage(
      'Flip CLI not found. Please install flip and ensure it\'s in your PATH.',
      'Learn More'
    ).then((selection) => {
      if (selection === 'Learn More') {
        vscode.env.openExternal(vscode.Uri.parse('https://github.com/httrp/flip'));
      }
    });
  }
}

async function updateStatusBar(): Promise<void> {
  if (!statusBarItem) {
    return;
  }

  const client = getFlipClient();
  const info = await client.getInfo();

  if (info.success && info.data) {
    const brain = info.data.active_brain || 'No brain';
    statusBarItem.text = `$(brain) ${brain}`;
    statusBarItem.show();
  } else {
    statusBarItem.text = '$(brain) Flip';
    statusBarItem.show();
  }
}

export function deactivate() {
  if (statusBarItem) {
    statusBarItem.dispose();
  }
}
