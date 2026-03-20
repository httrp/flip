import * as vscode from 'vscode';
import { getFlipClient, BrainInfo } from '../flip-client';

let output: vscode.OutputChannel | undefined;
function getOutput(): vscode.OutputChannel {
  if (!output) { output = vscode.window.createOutputChannel('Flip Restore'); }
  return output;
}
export function disposeRestoreOutput(): void { output?.dispose(); output = undefined; }

export async function brainRestore(): Promise<void> {
  const client = getFlipClient();

  const brainsResult = await client.getBrains();
  if (!brainsResult.success || !brainsResult.data) {
    vscode.window.showErrorMessage('Keine Brains gefunden');
    return;
  }

  const brains = brainsResult.data;
  if (brains.length === 0) {
    vscode.window.showErrorMessage('Keine Brains im Workspace');
    return;
  }

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains.map((b) => ({
    label: `$(database) ${b.name}`,
    description: b.active ? '(aktiv)' : '',
    detail: b.path,
    brain: b,
  }));

  brainItems.sort((a, b) => {
    if (a.brain.active) return -1;
    if (b.brain.active) return 1;
    return 0;
  });

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Brain für Restore auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  const mode = await vscode.window.showQuickPick(
    [
      { label: 'Interaktiv', description: 'Wähle Dateien einzeln aus (CLI)', value: 'interactive' },
      { label: 'Automatisch', description: 'Restore alle Dateien mit Journal-Verlinkung', value: 'auto' },
    ],
    { placeHolder: 'Restore-Modus wählen' }
  );

  if (!mode) {
    return;
  }

  const cmdArgs = mode.value === 'interactive' 
    ? ['brain', 'restore', selectedBrain.brain.path]
    : ['brain', 'restore', selectedBrain.brain.path, '--auto'];

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Restore ${mode.value === 'auto' ? '(Automatisch)' : '(Interaktiv)'} für "${selectedBrain.brain.name}"...`,
      cancellable: false,
    },
    async () => {
      return await client.runCommand(cmdArgs);
    }
  );

  if (!result.success) {
    const errorMsg = result.error ? (typeof result.error === 'string' ? result.error : JSON.stringify(result.error)) : 'Unbekannter Fehler';
    vscode.window.showErrorMessage(`Restore fehlgeschlagen: ${errorMsg}`);
    return;
  }

  getOutput().clear();
  getOutput().appendLine(`📂 Restore: ${selectedBrain.brain.name}`);
  getOutput().appendLine(`Brain: ${selectedBrain.brain.path}`);
  getOutput().appendLine(`Modus: ${mode.label}`);
  getOutput().appendLine('');
  
  if (mode.value === 'auto') {
    getOutput().appendLine('✅ Restore abgeschlossen!');
    getOutput().appendLine('');
    getOutput().appendLine('Alle orphaned Dateien wurden wiederhergestellt:');
    getOutput().appendLine('• Zurück in ihre ursprünglichen Ordner (meetings/, notes/, etc.)');
    getOutput().appendLine('• Mit Journal-Verlinkung, wenn Datumsübereinstimmung gefunden');
    getOutput().appendLine('• Leere .orphaned/ Verzeichnisse bereinigt');
  } else {
    getOutput().appendLine('📋 Interaktiver Restore gestartet im Terminal');
    getOutput().appendLine('Wähle die Dateien aus, die du wiederherstellen möchtest.');
  }
  
  getOutput().show(true);

  vscode.window.showInformationMessage(
    `Restore für "${selectedBrain.brain.name}" abgeschlossen!`,
    'Output öffnen'
  ).then((choice) => {
    if (choice === 'Output öffnen') getOutput().show(true);
  });
}
