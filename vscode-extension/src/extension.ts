import * as vscode from 'vscode';
import { getFlipClient, refreshFlipClient, BrainInfo } from './flip-client';
import { createJournal } from './commands/journal';
import { createMeetingNote } from './commands/meeting';
import { createNote, createQuicknote } from './commands/note';
import { createTask, updateTaskStatus } from './commands/task';
import { searchTasks, showFrogTasks, showTodaysTasks } from './commands/task-search';
import { searchAllBrains, showRecentNotes, searchWithFilters } from './commands/search';
import { syncBrains, quickSync, syncAndPush } from './commands/sync';
import { showStatus, switchBrain } from './commands/status';
import { insertLink, addCurrentFileToJournal } from './commands/insert-link';
import { addParticipantsCommand } from './commands/add-participants';
import { journalSyncCheck } from './commands/journal-sync';
import { brainHealthCheck } from './commands/brain-health';
import { mediaNormalize } from './commands/media-normalize';

let statusBarItem: vscode.StatusBarItem | undefined;

export function activate(context: vscode.ExtensionContext) {
  console.log('Flip extension activated');

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand('flip.journal', createJournal),
    vscode.commands.registerCommand('flip.note', createNote),
    vscode.commands.registerCommand('flip.quicknote', createQuicknote),
    vscode.commands.registerCommand('flip.taskNew', createTask),
    vscode.commands.registerCommand('flip.taskStatus', updateTaskStatus),
    vscode.commands.registerCommand('flip.taskSearch', searchTasks),
    vscode.commands.registerCommand('flip.taskFrog', showFrogTasks),
    vscode.commands.registerCommand('flip.taskToday', showTodaysTasks),
    vscode.commands.registerCommand('flip.status', showStatus),
    vscode.commands.registerCommand('flip.switchBrain', switchBrain),
    vscode.commands.registerCommand('flip.insertLink', insertLink),
    vscode.commands.registerCommand('flip.addToJournal', addCurrentFileToJournal),
    
    // Search commands
    vscode.commands.registerCommand('flip.search', searchAllBrains),
    vscode.commands.registerCommand('flip.recentNotes', showRecentNotes),
    vscode.commands.registerCommand('flip.searchFiltered', searchWithFilters),
    
    // Sync commands
    vscode.commands.registerCommand('flip.sync', syncBrains),
    vscode.commands.registerCommand('flip.quickSync', quickSync),
    vscode.commands.registerCommand('flip.syncAndPush', syncAndPush),
    
    // Participant management
    vscode.commands.registerCommand('flip.addParticipants', addParticipantsCommand),
    
    // Journal sync
    vscode.commands.registerCommand('flip.journalSync', journalSyncCheck),

    // Brain health
    vscode.commands.registerCommand('flip.brainHealth', brainHealthCheck),
    
    // Media normalize
    vscode.commands.registerCommand('flip.mediaNormalize', mediaNormalize),
    
    // Placeholder commands (to be implemented)
    vscode.commands.registerCommand('flip.meetingNote', createMeetingNote),
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
