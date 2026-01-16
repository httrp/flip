import * as vscode from 'vscode';
import { TaskBrowserPanel } from '../ui/taskBrowser';

/**
 * Open the task browser panel
 */
export async function browseTasksPanel(extensionUri: vscode.Uri): Promise<void> {
  TaskBrowserPanel.createOrShow(extensionUri);
}
