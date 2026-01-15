import * as vscode from 'vscode';
import { getFlipClient, DefinitionItem, DefinitionsResult } from '../flip-client';

/**
 * Definition item for quick pick
 */
interface DefinitionPickItem extends vscode.QuickPickItem {
  type: 'organization' | 'person' | 'context' | 'project';
  abbreviation: string;
  name: string;
}

/**
 * Show and manage definitions
 */
export async function definitionsCommand() {
  const client = getFlipClient();

  // Determine brain from active file
  const editor = vscode.window.activeTextEditor;
  const brainName = editor ? await client.getBrainForPath(editor.document.fileName) : undefined;

  // Get definitions
  const result = await client.listDefinitions(brainName);
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Fehler beim Laden der Definitionen: ${result.error}`);
    return;
  }

  const defs = result.data;

  // Build action menu
  const actions = [
    { label: '$(organization) Organisationen', description: `${defs.organizations.length} definiert`, action: 'orgs' },
    { label: '$(person) Personen', description: `${defs.people.length} definiert`, action: 'people' },
    { label: '$(tag) Kontexte', description: `${defs.contexts.length} definiert`, action: 'contexts' },
    { label: '$(add) Neue Organisation hinzufügen', action: 'add-org' },
    { label: '$(add) Neue Person hinzufügen', action: 'add-person' },
  ];

  const selected = await vscode.window.showQuickPick(actions, {
    placeHolder: 'Definitions verwalten',
  });

  if (!selected) {
    return;
  }

  switch (selected.action) {
    case 'orgs':
      await showDefinitionsList(client, 'organization', defs.organizations, brainName);
      break;
    case 'people':
      await showDefinitionsList(client, 'person', defs.people, brainName);
      break;
    case 'contexts':
      await showDefinitionsList(client, 'context', defs.contexts, brainName);
      break;
    case 'add-org':
      await addOrganization(client, brainName);
      break;
    case 'add-person':
      await addPerson(client, brainName);
      break;
  }
}

/**
 * Show list of definitions of a specific type
 */
async function showDefinitionsList(
  client: ReturnType<typeof getFlipClient>,
  type: 'organization' | 'person' | 'context' | 'project',
  items: DefinitionItem[],
  brainName?: string
) {
  if (items.length === 0) {
    const typeNames: Record<string, string> = {
      organization: 'Organisationen',
      person: 'Personen',
      context: 'Kontexte',
      project: 'Projekte',
    };
    vscode.window.showInformationMessage(`Keine ${typeNames[type]} definiert`);
    return;
  }

  const pickItems: DefinitionPickItem[] = items.map(item => ({
    label: `[${item.abbreviation}] ${item.name}`,
    description: item.description || item.organization || '',
    type,
    abbreviation: item.abbreviation,
    name: item.name,
  }));

  const selected = await vscode.window.showQuickPick(pickItems, {
    placeHolder: `${items.length} Einträge`,
  });

  if (!selected) {
    return;
  }

  // Show actions for selected item
  const actions = [
    { label: '$(eye) Details anzeigen', action: 'view' },
    { label: '$(edit) Bearbeiten', action: 'edit' },
    { label: '$(trash) Löschen', action: 'delete' },
  ];

  const action = await vscode.window.showQuickPick(actions, {
    placeHolder: `${selected.name} (${selected.abbreviation})`,
  });

  if (!action) {
    return;
  }

  switch (action.action) {
    case 'view':
      await showDefinitionDetails(selected);
      break;
    case 'edit':
      vscode.window.showInformationMessage('Bearbeiten: Bitte direkt in der YAML-Datei anpassen');
      // Open definitions file
      const pathResult = await client.getDefinitionsPath();
      if (pathResult.success && pathResult.data?.path) {
        const doc = await vscode.workspace.openTextDocument(pathResult.data.path);
        await vscode.window.showTextDocument(doc);
      }
      break;
    case 'delete':
      await deleteDefinition(client, selected, brainName);
      break;
  }
}

/**
 * Show details of a definition
 */
async function showDefinitionDetails(item: DefinitionPickItem) {
  const details = [
    `**Name:** ${item.name}`,
    `**Kürzel:** ${item.abbreviation}`,
    item.description ? `**Beschreibung:** ${item.description}` : '',
  ].filter(Boolean).join('\n\n');

  // Show in info message (simple for now)
  vscode.window.showInformationMessage(`[${item.abbreviation}] ${item.name}`);
}

/**
 * Delete a definition
 */
async function deleteDefinition(
  client: ReturnType<typeof getFlipClient>,
  item: DefinitionPickItem,
  brainName?: string
) {
  const confirm = await vscode.window.showWarningMessage(
    `"${item.name}" (${item.abbreviation}) wirklich löschen?`,
    { modal: true },
    'Löschen'
  );

  if (confirm !== 'Löschen') {
    return;
  }

  const result = await client.removeDefinition({
    type: item.type,
    abbreviation: item.abbreviation,
    brain: brainName,
  });

  if (result.success) {
    vscode.window.showInformationMessage(`${item.name} gelöscht`);
  } else {
    vscode.window.showErrorMessage(`Fehler: ${result.error}`);
  }
}

/**
 * Add a new organization
 */
async function addOrganization(client: ReturnType<typeof getFlipClient>, brainName?: string) {
  const abbr = await vscode.window.showInputBox({
    prompt: 'Kürzel der Organisation',
    placeHolder: 'z.B. ACME',
    validateInput: (value) => {
      if (!value || value.length === 0) {
        return 'Kürzel ist erforderlich';
      }
      if (!/^[A-Za-z0-9]+$/.test(value)) {
        return 'Nur Buchstaben und Zahlen erlaubt';
      }
      return null;
    },
  });

  if (!abbr) {
    return;
  }

  const name = await vscode.window.showInputBox({
    prompt: 'Vollständiger Name (optional)',
    placeHolder: 'z.B. Acme Corporation',
    value: abbr,
  });

  const description = await vscode.window.showInputBox({
    prompt: 'Beschreibung (optional)',
    placeHolder: 'z.B. Kunde seit 2024',
  });

  const result = await client.addOrganization({
    abbreviation: abbr.toUpperCase(),
    name: name || abbr,
    description: description || undefined,
    brain: brainName,
  });

  if (result.success) {
    vscode.window.showInformationMessage(`Organisation [${abbr.toUpperCase()}] erstellt`);
  } else {
    vscode.window.showErrorMessage(`Fehler: ${result.error}`);
  }
}

/**
 * Add a new person
 */
async function addPerson(client: ReturnType<typeof getFlipClient>, brainName?: string) {
  const name = await vscode.window.showInputBox({
    prompt: 'Name der Person',
    placeHolder: 'z.B. Max Mustermann',
  });

  if (!name) {
    return;
  }

  const abbr = await vscode.window.showInputBox({
    prompt: 'Kürzel (optional, wird automatisch generiert)',
    placeHolder: 'z.B. MM',
  });

  const org = await vscode.window.showInputBox({
    prompt: 'Organisation (optional)',
    placeHolder: 'z.B. ACME',
  });

  const role = await vscode.window.showInputBox({
    prompt: 'Rolle (optional)',
    placeHolder: 'z.B. Project Manager',
  });

  const result = await client.addPerson({
    name,
    abbreviation: abbr || undefined,
    organization: org || undefined,
    role: role || undefined,
    brain: brainName,
  });

  if (result.success && result.data) {
    vscode.window.showInformationMessage(`Person [${result.data.abbreviation}] ${name} erstellt`);
  } else {
    vscode.window.showErrorMessage(`Fehler: ${result.error}`);
  }
}

/**
 * Quick add organization command
 */
export async function addOrganizationCommand() {
  const client = getFlipClient();
  const editor = vscode.window.activeTextEditor;
  const brainName = editor ? await client.getBrainForPath(editor.document.fileName) : undefined;
  await addOrganization(client, brainName);
}

/**
 * Quick add person command
 */
export async function addPersonCommand() {
  const client = getFlipClient();
  const editor = vscode.window.activeTextEditor;
  const brainName = editor ? await client.getBrainForPath(editor.document.fileName) : undefined;
  await addPerson(client, brainName);
}
