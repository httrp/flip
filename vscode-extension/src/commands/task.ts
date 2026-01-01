import * as vscode from 'vscode';
import { getFlipClient, TaskResult } from '../flip-client';

/**
 * Create a new task with VS Code input dialogs
 */
export async function createTask(): Promise<void> {
  const client = getFlipClient();

  // Get description
  const description = await vscode.window.showInputBox({
    prompt: 'Task description',
    placeHolder: 'What needs to be done?',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Description is required';
      }
      return null;
    },
  });

  if (!description) {
    return; // User cancelled
  }

  // Get due date (optional)
  const dueOptions = [
    { label: 'No due date', value: '' },
    { label: 'Today', value: 'today' },
    { label: 'Tomorrow', value: 'tomorrow' },
    { label: 'Custom date...', value: 'custom' },
  ];

  const selectedDue = await vscode.window.showQuickPick(dueOptions, {
    placeHolder: 'Due date (optional)',
  });

  let due: string | undefined;
  if (selectedDue?.value === 'custom') {
    const customDate = await vscode.window.showInputBox({
      prompt: 'Due date (YYYY-MM-DD)',
      placeHolder: '2025-01-15',
      validateInput: (value) => {
        if (value && !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
          return 'Please use YYYY-MM-DD format';
        }
        return null;
      },
    });
    due = customDate;
  } else if (selectedDue?.value) {
    due = selectedDue.value;
  }

  // Get priority (optional)
  const priorityOptions = [
    { label: '$(arrow-up) High', value: 'high' },
    { label: '$(dash) Medium', value: 'medium' },
    { label: '$(arrow-down) Low', value: 'low' },
    { label: 'No priority', value: '' },
  ];

  const selectedPriority = await vscode.window.showQuickPick(priorityOptions, {
    placeHolder: 'Priority (optional)',
  });

  const priority = selectedPriority?.value;

  // Get brain selection if multiple brains
  const info = await client.getInfo();
  let brain: string | undefined;

  if (info.success && info.data && info.data.brains.length > 1) {
    const brainItems = info.data.brains.map((b) => ({
      label: b.name,
      description: b.active ? '(active)' : '',
      detail: `${b.type} - ${b.path}`,
    }));

    const selected = await vscode.window.showQuickPick(brainItems, {
      placeHolder: 'Select brain for task',
    });

    if (selected) {
      brain = selected.label;
    }
  }

  // Create task
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating task...',
      cancellable: false,
    },
    async () => {
      const result = await client.createTask({
        description: description.trim(),
        brain,
        due,
        priority,
      });

      if (!result.success) {
        vscode.window.showErrorMessage(`Failed to create task: ${result.error}`);
        return;
      }

      const data = result.data as TaskResult;

      // Open the file
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      let msg = `Created task in ${data.brain_name}`;
      if (data.due) {
        msg += ` (due: ${data.due})`;
      }
      vscode.window.showInformationMessage(msg);
    }
  );
}
