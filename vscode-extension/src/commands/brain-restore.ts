import * as vscode from 'vscode';
import { getFlipClient, BrainInfo } from '../flip-client';

const output = vscode.window.createOutputChannel('Flip Restore');

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

  output.clear();
  output.appendLine(`📂 Restore: ${selectedBrain.brain.name}`);
  output.appendLine(`Brain: ${selectedBrain.brain.path}`);
  output.appendLine(`Modus: ${mode.label}`);
  output.appendLine('');
  
  if (mode.value === 'auto') {
    output.appendLine('✅ Restore abgeschlossen!');
    output.appendLine('');
    output.appendLine('Alle orphaned Dateien wurden wiederhergestellt:');
    output.appendLine('• Zurück in ihre ursprünglichen Ordner (meetings/, notes/, etc.)');
    output.appendLine('• Mit Journal-Verlinkung, wenn Datumsübereinstimmung gefunden');
    output.appendLine('• Leere .orphaned/ Verzeichnisse bereinigt');
  } else {
    output.appendLine('📋 Interaktiver Restore gestartet im Terminal');
    output.appendLine('Wähle die Dateien aus, die du wiederherstellen möchtest.');
  }
  
  output.show(true);

  vscode.window.showInformationMessage(
    `Restore für "${selectedBrain.brain.name}" abgeschlossen!`,
    'Output öffnen'
  ).then((choice) => {
    if (choice === 'Output öffnen') output.show(true);
  });
}
