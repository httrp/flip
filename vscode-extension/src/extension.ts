import * as vscode from 'vscode';
import { getFlipClient, refreshFlipClient, BrainInfo } from './flip-client';
import { createJournal, insertTasksIntoJournalCommand } from './commands/journal';
import { createMeetingNote } from './commands/meeting';
import { createNote, createQuicknote } from './commands/note';
import { createTask, updateTaskStatus, markTaskDone } from './commands/task';
import { searchTasks, showFrogTasks, showTodaysTasks } from './commands/task-search';
import { browseTasksPanel } from './commands/task-browse';
import { searchAllBrains, showRecentNotes, searchWithFilters } from './commands/search';
import { syncBrains, quickSync, syncAndPush } from './commands/sync';
import { showStatus, switchBrain } from './commands/status';
import { insertLink, addCurrentFileToJournal } from './commands/insert-link';
import { addParticipantsCommand } from './commands/add-participants';
import { journalSyncCheck } from './commands/journal-sync';
import { brainHealthCheck } from './commands/brain-health';
import { brainRestore } from './commands/brain-restore';
import { brainRelocate } from './commands/brain-relocate';
import { mediaNormalize } from './commands/media-normalize';
import { aiStatusCommand, aiResearchCommand, aiSummarizeCommand, aiImproveCommand } from './commands/ai';
import { definitionsCommand, addOrganizationCommand, addPersonCommand } from './commands/definitions';
import { exerciseCommand, newExerciseCommand, quickTrackCommand } from './commands/exercises';
import { templateCommand, editTemplateCommand, resetTemplateCommand } from './commands/templates';
import { convertToFlipNote } from './commands/convert-to-note';
import { showAbout, showHelp } from './commands/about';
import { manageBrains, listBrains, addBrain, initBrain, removeBrain, setDefaultBrain } from './commands/brain';
import { showManage, showSettings } from './commands/manage';

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
    vscode.commands.registerCommand('flip.taskBrowse', () => browseTasksPanel(context.extensionUri)),
    vscode.commands.registerCommand('flip.status', showStatus),
    vscode.commands.registerCommand('flip.switchBrain', switchBrain),
    vscode.commands.registerCommand('flip.aiStatus', aiStatusCommand),
    vscode.commands.registerCommand('flip.aiResearch', aiResearchCommand),
    vscode.commands.registerCommand('flip.aiSummarize', aiSummarizeCommand),
    vscode.commands.registerCommand('flip.aiImprove', aiImproveCommand),
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
    vscode.commands.registerCommand('flip.brainRestore', brainRestore),
    vscode.commands.registerCommand('flip.brainRelocate', brainRelocate),
    
    // Brain management
    vscode.commands.registerCommand('flip.manageBrains', manageBrains),
    vscode.commands.registerCommand('flip.listBrains', listBrains),
    vscode.commands.registerCommand('flip.addBrain', addBrain),
    vscode.commands.registerCommand('flip.initBrain', initBrain),
    vscode.commands.registerCommand('flip.removeBrain', removeBrain),
    vscode.commands.registerCommand('flip.setDefaultBrain', setDefaultBrain),
    
    // Manage & Settings
    vscode.commands.registerCommand('flip.manage', showManage),
    vscode.commands.registerCommand('flip.settings', showSettings),
    
    // Media normalize
    vscode.commands.registerCommand('flip.mediaNormalize', mediaNormalize),
    
    // Definitions management
    vscode.commands.registerCommand('flip.definitions', definitionsCommand),
    vscode.commands.registerCommand('flip.addOrganization', addOrganizationCommand),
    vscode.commands.registerCommand('flip.addPerson', addPersonCommand),
    
    // Exercise management
    vscode.commands.registerCommand('flip.exercises', exerciseCommand),
    vscode.commands.registerCommand('flip.newExercise', newExerciseCommand),
    vscode.commands.registerCommand('flip.quickTrack', quickTrackCommand),
    
    // Template management
    vscode.commands.registerCommand('flip.templates', templateCommand),
    vscode.commands.registerCommand('flip.editTemplate', editTemplateCommand),
    vscode.commands.registerCommand('flip.resetTemplate', resetTemplateCommand),
    
    // Journal task insertion
    vscode.commands.registerCommand('flip.insertTasks', insertTasksIntoJournalCommand),
    
    // Help & About
    vscode.commands.registerCommand('flip.about', showAbout),
    vscode.commands.registerCommand('flip.help', showHelp),
    
    // Convert file to flip note
    vscode.commands.registerCommand('flip.convertToNote', convertToFlipNote),
    
    // Placeholder commands (to be implemented)
    vscode.commands.registerCommand('flip.meetingNote', createMeetingNote),
    vscode.commands.registerCommand('flip.taskDone', markTaskDone)
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
