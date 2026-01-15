import * as vscode from 'vscode';
import { getFlipClient, FlipResult, JournalResult, BrainInfo, TaskInfo } from '../flip-client';

// Task filter options for journal insertion
const TASK_FILTERS = [
  { label: '$(warning) Overdue Tasks', value: 'overdue', description: 'Tasks past their due date' },
  { label: '$(calendar) Due Today', value: 'today', description: 'Tasks due today' },
  { label: '$(flame) Frogs (Important)', value: 'frog', description: '🐸 Most important tasks' },
  { label: '$(checklist) All Open Tasks', value: 'all', description: 'All open tasks' },
  { label: '$(x) Skip', value: 'skip', description: 'Don\'t add tasks' }
];

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
  let journalData: JournalResult | undefined;
  let wasCreated = false;
  
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

      journalData = result.data as JournalResult;
      wasCreated = journalData.action === 'created';

      // Open the file
      const doc = await vscode.workspace.openTextDocument(journalData.path);
      await vscode.window.showTextDocument(doc);

      // Show notification
      const action = wasCreated ? 'Created' : 'Opened';
      vscode.window.showInformationMessage(
        `${action} journal for ${journalData.date} in ${journalData.brain_name}`
      );
    }
  );

  // If journal was newly created, ask about inserting tasks
  if (wasCreated && journalData) {
    await askToInsertTasks(journalData);
  }
}

/**
 * Ask user if they want to insert tasks into the journal
 */
async function askToInsertTasks(journalData: JournalResult): Promise<void> {
  const selected = await vscode.window.showQuickPick(TASK_FILTERS, {
    placeHolder: 'Add tasks to your journal?',
    title: 'Insert Tasks'
  });

  if (!selected || selected.value === 'skip') {
    return;
  }

  await insertTasksIntoJournal(selected.value, journalData.brain_name);
}

/**
 * Insert tasks into the current journal (can be called separately)
 */
export async function insertTasksIntoJournalCommand(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showErrorMessage('No active editor');
    return;
  }

  // Check if current file looks like a journal
  const fileName = editor.document.fileName;
  if (!fileName.includes('journal') && !fileName.match(/\d{4}-\d{2}-\d{2}/)) {
    const confirm = await vscode.window.showWarningMessage(
      'Current file doesn\'t look like a journal. Insert tasks anyway?',
      'Yes', 'No'
    );
    if (confirm !== 'Yes') {
      return;
    }
  }

  const selected = await vscode.window.showQuickPick(
    TASK_FILTERS.filter(f => f.value !== 'skip'),
    {
      placeHolder: 'Which tasks to insert?',
      title: 'Insert Tasks into Journal'
    }
  );

  if (!selected) {
    return;
  }

  await insertTasksIntoJournal(selected.value);
}

/**
 * Insert filtered tasks into the active editor
 */
async function insertTasksIntoJournal(filterType: string, brainName?: string): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    return;
  }

  const client = getFlipClient();
  const today = new Date().toISOString().split('T')[0];

  // Fetch tasks based on filter
  let tasks: TaskInfo[] = [];
  
  const result = await client.getTasks({ 
    status: 'open',
    brain: brainName 
  });

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage('Failed to fetch tasks');
    return;
  }

  const allTasks = result.data.tasks;

  switch (filterType) {
    case 'overdue':
      tasks = allTasks.filter(t => t.due && t.due < today);
      break;
    case 'today':
      tasks = allTasks.filter(t => t.due === today);
      break;
    case 'frog':
      tasks = allTasks.filter(t => t.frog);
      break;
    case 'all':
      tasks = allTasks;
      break;
  }

  if (tasks.length === 0) {
    vscode.window.showInformationMessage(`No ${filterType} tasks found`);
    return;
  }

  // Format tasks as links
  const taskSection = formatTasksSection(tasks, filterType);

  // Find a good insertion point (after ## Tasks or at end of file)
  const document = editor.document;
  const text = document.getText();
  
  let insertPosition: vscode.Position;
  
  // Look for existing Tasks section
  const tasksMatch = text.match(/^## (?:Tasks|Offene Tasks|Open Tasks|New Tasks)/m);
  if (tasksMatch && tasksMatch.index !== undefined) {
    // Find end of line after section header
    const lineStart = document.positionAt(tasksMatch.index);
    insertPosition = new vscode.Position(lineStart.line + 1, 0);
  } else {
    // Insert at end of file
    insertPosition = new vscode.Position(document.lineCount, 0);
  }

  // Insert the task links
  await editor.edit(editBuilder => {
    editBuilder.insert(insertPosition, taskSection);
  });

  vscode.window.showInformationMessage(`Inserted ${tasks.length} task links`);
}

/**
 * Format tasks as a markdown section with links
 */
function formatTasksSection(tasks: TaskInfo[], filterType: string): string {
  const lines: string[] = [];
  
  // Add section header if inserting at end
  const headerMap: Record<string, string> = {
    'overdue': '### ⚠️ Overdue Tasks',
    'today': '### 📅 Due Today',
    'frog': '### 🐸 Frogs',
    'all': '### 📋 Open Tasks'
  };
  
  lines.push('');
  lines.push(headerMap[filterType] || '### Tasks');
  lines.push('');

  for (const task of tasks) {
    const icons: string[] = [];
    if (task.frog) icons.push('🐸');
    if (task.priority === 'high') icons.push('⏫');
    if (task.due) icons.push(`📅 ${task.due}`);
    
    const prefix = icons.length > 0 ? icons.join(' ') + ' ' : '';
    
    // Create a link to the task file
    // Format: - [ ] [Task description](path/to/task.md)
    const relPath = task.rel_path || task.path;
    lines.push(`- [ ] ${prefix}[${task.description}](${relPath})`);
  }

  lines.push('');
  return lines.join('\n');
}

