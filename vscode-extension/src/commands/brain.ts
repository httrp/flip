import * as vscode from 'vscode';
import { getFlipClient, BrainInfo } from '../flip-client';

/**
 * List all brains in the workspace
 */
export async function listBrains(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getBrains();
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage('Fehler beim Abrufen der Brains');
    return;
  }

  const brains = result.data;
  if (brains.length === 0) {
    vscode.window.showInformationMessage('Keine Brains im Workspace gefunden');
    return;
  }

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains.map((b) => ({
    label: `${b.active ? '$(debug-breakpoint-conditional-unverified)' : '$(circle-outline)'} ${b.name}`,
    description: b.active ? 'Aktiv' : '',
    detail: b.path,
    brain: b,
  }));

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Brain auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  // Show brain details in information message
  const message = `Brain: ${selectedBrain.brain.name}\nPath: ${selectedBrain.brain.path}\nAktiv: ${
    selectedBrain.brain.active ? 'Ja' : 'Nein'
  }`;
  vscode.window.showInformationMessage(message);
}

/**
 * Add an existing brain to the workspace
 */
export async function addBrain(): Promise<void> {
  const client = getFlipClient();

  // Get brain path
  const pathInput = await vscode.window.showInputBox({
    prompt: 'Pfad zum Brain eingeben',
    placeHolder: '/path/to/brain',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Pfad ist erforderlich';
      }
      return null;
    },
  });

  if (!pathInput) {
    return;
  }

  // Get optional name
  const name = await vscode.window.showInputBox({
    prompt: 'Name für das Brain (optional)',
    placeHolder: 'Leer lassen für automatischen Namen',
  });

  // Ask if should be set as default
  const setDefault = await vscode.window.showQuickPick(
    [
      { label: 'Ja', description: 'Als Standard-Brain setzen', value: true },
      { label: 'Nein', value: false },
    ],
    { placeHolder: 'Als Standard-Brain setzen?' }
  );

  if (setDefault === undefined) {
    return;
  }

  // Call flip brain add command
  const result = await client.runCommand([
    'brain',
    'add',
    pathInput,
    ...(name ? ['--name', name] : []),
    ...(setDefault ? ['--default'] : []),
  ]);

  if (result.success) {
    vscode.window.showInformationMessage(`Brain hinzugefügt: ${name || pathInput}`);
    // Refresh client to update brain list
    const { refreshFlipClient } = await import('../flip-client');
    refreshFlipClient();
  } else {
    vscode.window.showErrorMessage(`Fehler beim Hinzufügen des Brain: ${result.error}`);
  }
}

/**
 * Initialize a new brain
 */
export async function initBrain(): Promise<void> {
  const client = getFlipClient();

  // Ask: init existing directory or create new?
  const mode = await vscode.window.showQuickPick(
    [
      { label: '$(folder-opened) Existierendes Verzeichnis initialisieren', value: 'existing' },
      { label: '$(new-folder) Neues Brain-Verzeichnis erstellen', value: 'new' },
    ],
    { placeHolder: 'Brain erstellen oder bestehendes Verzeichnis initialisieren?' }
  );

  if (!mode) {
    return;
  }

  let targetPath: string | undefined;

  if (mode.value === 'existing') {
    // Let user pick existing folder
    const folderUri = await vscode.window.showOpenDialog({
      canSelectFolders: true,
      canSelectFiles: false,
      canSelectMany: false,
      openLabel: 'Verzeichnis als Brain initialisieren',
    });

    if (!folderUri || folderUri.length === 0) {
      return;
    }
    targetPath = folderUri[0].fsPath;
  } else {
    // Ask for path
    targetPath = await vscode.window.showInputBox({
      prompt: 'Pfad für das neue Brain',
      placeHolder: 'z.B. ~/universe/pr/my-brain',
    });
  }

  if (!targetPath) {
    return;
  }

  // Get brain name
  const defaultName = targetPath.split('/').pop() || 'brain';
  const name = await vscode.window.showInputBox({
    prompt: 'Name des Brain',
    value: defaultName,
    placeHolder: 'z.B. "pinky"',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Name ist erforderlich';
      }
      return null;
    },
  });

  if (!name) {
    return;
  }

  // Ask for template
  const template = await vscode.window.showQuickPick(
    [
      { label: 'Personal', description: 'Persönliches Brain', value: 'personal' },
      { label: 'Work', description: 'Arbeits-Brain', value: 'work' },
      { label: 'Learning', description: 'Lern-Brain', value: 'learning' },
    ],
    { placeHolder: 'Template auswählen' }
  );

  if (!template) {
    return;
  }

  // Call flip brain init
  const result = await client.runCommand([
    'brain',
    'init',
    targetPath,
    '--name',
    name,
    '--template',
    template.value,
    '--force',
  ]);

  if (result.success) {
    vscode.window.showInformationMessage(`Brain initialisiert: ${name}`);
    // Refresh client to update brain list
    const { refreshFlipClient } = await import('../flip-client');
    refreshFlipClient();
  } else {
    vscode.window.showErrorMessage(`Fehler beim Initialisieren des Brain: ${result.error}`);
  }
}

/**
 * Remove a brain from the workspace
 */
export async function removeBrain(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getBrains();
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage('Fehler beim Abrufen der Brains');
    return;
  }

  const brains = result.data;
  if (brains.length === 0) {
    vscode.window.showErrorMessage('Keine Brains zum Löschen vorhanden');
    return;
  }

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains
    .filter((b) => !b.active) // Don't allow removing the active brain
    .map((b) => ({
      label: `$(database) ${b.name}`,
      detail: b.path,
      brain: b,
    }));

  if (brainItems.length === 0) {
    vscode.window.showErrorMessage('Kann nur nicht-aktive Brains löschen');
    return;
  }

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Brain zum Löschen auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  // Confirm deletion
  const confirmed = await vscode.window.showWarningMessage(
    `Brain "${selectedBrain.brain.name}" wirklich entfernen? (Dateien bleiben erhalten)`,
    'Ja, entfernen',
    'Abbrechen'
  );

  if (confirmed !== 'Ja, entfernen') {
    return;
  }

  // Call flip brain remove command
  const removeResult = await client.runCommand(['brain', 'remove', selectedBrain.brain.name]);

  if (removeResult.success) {
    vscode.window.showInformationMessage(`Brain entfernt: ${selectedBrain.brain.name}`);
    // Refresh client to update brain list
    const { refreshFlipClient } = await import('../flip-client');
    refreshFlipClient();
  } else {
    vscode.window.showErrorMessage(`Fehler beim Löschen des Brain: ${removeResult.error}`);
  }
}

/**
 * Set a brain as default for the workspace
 */
export async function setDefaultBrain(): Promise<void> {
  const client = getFlipClient();

  const result = await client.getBrains();
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage('Fehler beim Abrufen der Brains');
    return;
  }

  const brains = result.data;
  if (brains.length <= 1) {
    vscode.window.showInformationMessage('Nur ein Brain vorhanden');
    return;
  }

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains.map((b) => ({
    label: `${b.active ? '$(debug-breakpoint-conditional-unverified)' : '$(circle-outline)'} ${b.name}`,
    description: b.active ? 'Derzeit aktiv' : '',
    detail: b.path,
    brain: b,
  }));

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Neues Standard-Brain auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  // Call flip brain set-default command
  const setResult = await client.runCommand(['brain', 'set-default', selectedBrain.brain.name]);

  if (setResult.success) {
    vscode.window.showInformationMessage(`Standard-Brain gesetzt: ${selectedBrain.brain.name}`);
    // Refresh client to update brain list
    const { refreshFlipClient } = await import('../flip-client');
    refreshFlipClient();
  } else {
    vscode.window.showErrorMessage(`Fehler beim Setzen des Standard-Brain: ${setResult.error}`);
  }
}

/**
 * Show brain management menu
 */
export async function manageBrains(): Promise<void> {
  const actions = await vscode.window.showQuickPick(
    [
      { label: '$(list-unordered) Liste anzeigen', value: 'list', description: 'Alle Brains auflisten' },
      { label: '$(plus) Brain hinzufügen', value: 'add', description: 'Existierendes Brain hinzufügen' },
      { label: '$(file-add) Brain erstellen', value: 'init', description: 'Neues Brain erstellen' },
      { label: '$(close) Brain entfernen', value: 'remove', description: 'Brain entfernen' },
      { label: '$(star-full) Standard-Brain setzen', value: 'default', description: 'Standard-Brain ändern' },
    ],
    { placeHolder: 'Was möchtest du tun?' }
  );

  if (!actions) {
    return;
  }

  switch (actions.value) {
    case 'list':
      await listBrains();
      break;
    case 'add':
      await addBrain();
      break;
    case 'init':
      await initBrain();
      break;
    case 'remove':
      await removeBrain();
      break;
    case 'default':
      await setDefaultBrain();
      break;
  }
}
