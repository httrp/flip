import * as vscode from 'vscode';
import * as path from 'path';
import { getFlipClient, TaskInfo } from '../flip-client';

/**
 * Search for tasks and navigate to them
 */
export async function searchTasks(): Promise<void> {
  const client = getFlipClient();

  // Create a QuickPick with live search
  const quickPick = vscode.window.createQuickPick<TaskQuickPickItem>();
  quickPick.placeholder = 'Search tasks by description...';
  quickPick.matchOnDescription = true;
  quickPick.matchOnDetail = true;

  // Initial load - show open tasks
  quickPick.busy = true;
  const initialResult = await client.getTasks({ status: 'open' });
  if (initialResult.success && initialResult.data) {
    quickPick.items = initialResult.data.tasks.map(taskToQuickPickItem);
  }
  quickPick.busy = false;

  // Handle search input
  let searchTimeout: NodeJS.Timeout | undefined;
  quickPick.onDidChangeValue(async (value) => {
    // Debounce the search
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }
    searchTimeout = setTimeout(async () => {
      quickPick.busy = true;
      const result = await client.getTasks({ query: value });
      if (result.success && result.data) {
        quickPick.items = result.data.tasks.map(taskToQuickPickItem);
      }
      quickPick.busy = false;
    }, 200);
  });

  // Handle selection
  quickPick.onDidAccept(async () => {
    const selected = quickPick.selectedItems[0];
    if (selected && selected.task) {
      quickPick.hide();
      await showTaskActions(selected.task);
    }
  });

  quickPick.onDidHide(() => quickPick.dispose());
  quickPick.show();
}

/**
 * Show only frog tasks
 */
export async function showFrogTasks(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getTasks({ frog: true, status: 'open' });
  if (!result.success || !result.data || result.data.tasks.length === 0) {
    vscode.window.showInformationMessage('No 🐸 frog tasks found');
    return;
  }

  const items = result.data.tasks.map(taskToQuickPickItem);
  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: '🐸 Eat the Frog - Select your most important task',
    matchOnDescription: true,
    matchOnDetail: true,
  });

  if (selected && selected.task) {
    await showTaskActions(selected.task);
  }
}

/**
 * Show today's tasks
 */
export async function showTodaysTasks(): Promise<void> {
  const client = getFlipClient();

  // Get tasks due today
  const today = new Date().toISOString().split('T')[0];
  const result = await client.getTasks({ status: 'open' });
  
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage('Failed to fetch tasks');
    return;
  }

  // Filter for today's tasks
  const todayTasks = result.data.tasks.filter(
    (t) => t.due === today || t.frog
  );

  if (todayTasks.length === 0) {
    vscode.window.showInformationMessage('No tasks due today');
    return;
  }

  const items = todayTasks.map(taskToQuickPickItem);
  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: "📅 Today's tasks",
    matchOnDescription: true,
    matchOnDetail: true,
  });

  if (selected && selected.task) {
    await showTaskActions(selected.task);
  }
}

// Helper interfaces and functions

interface TaskQuickPickItem extends vscode.QuickPickItem {
  task?: TaskInfo;
}

function taskToQuickPickItem(task: TaskInfo): TaskQuickPickItem {
  const icons: string[] = [];
  
  if (task.frog) {
    icons.push('🐸');
  }
  if (task.priority === 'high') {
    icons.push('⏫');
  } else if (task.priority === 'medium') {
    icons.push('🔼');
  } else if (task.priority === 'low') {
    icons.push('🔽');
  }
  if (task.due) {
    icons.push(`📅 ${task.due}`);
  }
  if (task.status === 'in-progress') {
    icons.push('🔄');
  } else if (task.status === 'done') {
    icons.push('✅');
  }

  const prefix = icons.length > 0 ? icons.join(' ') + ' ' : '';
  const statusText = formatStatus(task.status);
  
  return {
    label: `${prefix}${task.description}`,
    description: `${statusText} · ${task.brain_name}`,
    detail: `${task.rel_path}:${task.line}` + (task.tags?.length ? ` · ${task.tags.join(' ')}` : ''),
    task,
  };
}

function formatStatus(status: string): string {
  switch (status) {
    case 'open':
      return '⬜ Open';
    case 'in-progress':
      return '🔄 In Progress';
    case 'done':
      return '✅ Done';
    case 'deferred':
      return '⏸ Deferred';
    case 'cancelled':
      return '❌ Cancelled';
    default:
      return `• ${status}`;
  }
}

async function openTaskLocation(task: TaskInfo): Promise<void> {
  try {
    const doc = await vscode.workspace.openTextDocument(task.path);
    const editor = await vscode.window.showTextDocument(doc);
    
    // Navigate to the task line
    const position = new vscode.Position(task.line - 1, 0);
    editor.selection = new vscode.Selection(position, position);
    editor.revealRange(
      new vscode.Range(position, position),
      vscode.TextEditorRevealType.InCenter
    );
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to open task: ${error}`);
  }
}

/**
 * Show task action menu after selection
 */
async function showTaskActions(task: TaskInfo): Promise<void> {
  const actions = [
    { label: '$(file-text) Open task', value: 'open' },
    { label: '$(book) Add to current journal', value: 'journal' },
    { label: '$(check) Mark as Done', value: 'done' },
    { label: '$(sync) Set In Progress', value: 'in-progress' },
    { label: '$(circle-outline) Set Open', value: 'open-status' },
    { label: '$(clock) Defer', value: 'deferred' },
    { label: '$(close) Cancel', value: 'cancelled' },
  ];

  const selected = await vscode.window.showQuickPick(actions, {
    placeHolder: `Action for: ${task.description}`,
  });

  if (!selected) {
    return;
  }

  if (selected.value === 'open') {
    await openTaskLocation(task);
  } else if (selected.value === 'journal') {
    await addTaskToCurrentJournal(task);
  } else {
    // Update status
    const client = getFlipClient();
    const status = selected.value === 'open-status' ? 'open' : selected.value as 'done' | 'in-progress' | 'deferred' | 'cancelled';
    
    const result = await client.updateTaskStatus({
      file: task.path,
      line: task.line,
      status,
    });

    if (result.success) {
      vscode.window.showInformationMessage(`✓ Task marked as ${status}`);
    } else {
      vscode.window.showErrorMessage(`Failed to update task: ${result.error}`);
    }
  }
}

async function addTaskToCurrentJournal(task: TaskInfo): Promise<void> {
  const client = getFlipClient();
  const activeEditor = vscode.window.activeTextEditor;
  let date: string | undefined;

  // If a journal file is currently open, link into that journal date.
  if (activeEditor) {
    const fileName = activeEditor.document.fileName;
    const basename = path.basename(fileName);
    const dateFromFilename = basename.match(/(\d{4}-\d{2}-\d{2})/);
    if (dateFromFilename) {
      date = dateFromFilename[1];
    } else {
      const firstLine = activeEditor.document.lineAt(0).text;
      const dateFromHeading = firstLine.match(/\b(\d{4}-\d{2}-\d{2})\b/);
      if (dateFromHeading) {
        date = dateFromHeading[1];
      }
    }
  }

  const result = await client.addToJournal({
    file: task.path,
    title: task.description,
    type: 'task',
    brain: task.brain_name,
    date,
  });

  if (!result.success) {
    vscode.window.showWarningMessage(`Failed to add task to journal: ${result.error}`);
    return;
  }

  const targetText = date ? `journal (${date})` : 'today\'s journal';
  vscode.window.showInformationMessage(`✓ Task linked in ${targetText}`);
}
