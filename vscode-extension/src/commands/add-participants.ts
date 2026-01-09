import * as vscode from 'vscode';
import { FlipClient } from '../flip-client';

/**
 * Participant item for quick pick
 */
interface ParticipantItem extends vscode.QuickPickItem {
  person: string;
  isFromSeries?: boolean;
}

/**
 * Add or update participants in a meeting note
 */
export async function addParticipantsCommand(client: FlipClient) {
  // Get active editor
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showErrorMessage('Bitte öffne zuerst eine Datei');
    return;
  }

  const filePath = editor.document.fileName;
  const content = editor.document.getText();

  // Get the current participants if any
  let currentParticipants: string[] = [];
  const participantsMatch = content.match(/^### Participants\n([\s\S]*?)(?=\n###|\Z)/m);
  if (participantsMatch) {
    const participantsText = participantsMatch[1].trim();
    if (participantsText && participantsText !== '-') {
      currentParticipants = participantsText
        .split('\n')
        .map(line => line.replace(/^-\s*/, '').trim())
        .filter(p => p.length > 0);
    }
  }

  // Get all definitions
  const defsResult = await client.listDefinitions();
  if (!defsResult.success || !defsResult.data) {
    vscode.window.showErrorMessage('Fehler beim Laden der Definitionen');
    return;
  }

  // Check if this is a series meeting
  const seriesMatch = content.match(/^- series:: (.+)$/m);
  let seriesParticipants: string[] = [];
  if (seriesMatch) {
    const seriesName = seriesMatch[1].trim();
    const seriesResult = await client.getSeriesParticipants(seriesName);
    if (seriesResult.success && seriesResult.data?.participants) {
      seriesParticipants = seriesResult.data.participants;
    }
  }

  // Build items for quick pick with checkboxes
  const items: ParticipantItem[] = [];
  const seenParticipants = new Set<string>();

  // Add historical series participants first (for series meetings)
  if (seriesParticipants.length > 0) {
    items.push({
      label: '── Aus Serie (historisch) ──',
      kind: vscode.QuickPickItemKind.Separator,
      person: '',
    });
    for (const p of seriesParticipants) {
      seenParticipants.add(p);
      const isSelected = currentParticipants.includes(p);
      items.push({
        label: `${isSelected ? '$(check)' : '$(circle-outline)'} ${p}`,
        person: p,
        isFromSeries: true,
        picked: isSelected,
        description: isSelected ? 'ausgewählt' : '',
      });
    }
  }

  // Add people from definitions
  if (defsResult.data.people.length > 0) {
    items.push({
      label: '── Definitionen ──',
      kind: vscode.QuickPickItemKind.Separator,
      person: '',
    });
    for (const person of defsResult.data.people) {
      const display = person.organization ? `${person.name} (${person.organization})` : person.name;
      if (!seenParticipants.has(display)) {
        const isSelected = currentParticipants.includes(person.name) || currentParticipants.includes(display);
        items.push({
          label: `${isSelected ? '$(check)' : '$(circle-outline)'} ${display}`,
          person: display,
          picked: isSelected,
          description: isSelected ? 'ausgewählt' : '',
        });
        seenParticipants.add(display);
      }
    }
  }

  // Add option to create new person
  items.push({
    label: '── Neue Person ──',
    kind: vscode.QuickPickItemKind.Separator,
    person: '',
  });
  items.push({
    label: '$(plus) Neue Person hinzufügen',
    person: '__new__',
  });

  // Show quick pick with canSelectMany
  const selected = await vscode.window.showQuickPick(items, {
    canPickMany: true,
    placeHolder: 'Wähle Teilnehmer aus (ENTER zum Bestätigen, ESC zum Abbrechen)',
    matchOnDescription: true,
  });

  if (selected === undefined) {
    return; // User cancelled
  }

  // Check if user wants to add new person
  const hasNewPerson = selected.some(item => item.person === '__new__');
  if (hasNewPerson) {
    // Remove __new__ from selection
    const withoutNew = selected.filter(item => item.person !== '__new__');
    await addNewPerson(client, withoutNew);
    return;
  }

  // Update participants in the file
  const participants = selected
    .map(item => item.person)
    .filter(p => p.length > 0);

  if (participants.length === 0) {
    vscode.window.showWarningMessage('Keine Teilnehmer ausgewählt');
    return;
  }

  updateParticipants(editor, participants);
  vscode.window.showInformationMessage(`${participants.length} Teilnehmer hinzugefügt`);
}

/**
 * Handle adding a new person
 */
async function addNewPerson(
  client: FlipClient,
  currentSelection: vscode.QuickPickItem[]
) {
  // Prompt for name
  const name = await vscode.window.showInputBox({
    prompt: 'Name der neuen Person',
    placeHolder: 'z.B. John Doe',
  });

  if (!name) {
    return;
  }

  // Prompt for organization
  const org = await vscode.window.showInputBox({
    prompt: 'Organisation (optional)',
    placeHolder: 'z.B. Acme Corp',
  });

  // Prompt for role
  const role = await vscode.window.showInputBox({
    prompt: 'Rolle (optional)',
    placeHolder: 'z.B. Project Manager',
  });

  // Add to definitions
  const result = await client.addPerson({
    name,
    organization: org || undefined,
    role: role || undefined,
  });

  if (!result.success) {
    vscode.window.showErrorMessage(`Fehler: ${result.error}`);
    return;
  }

  vscode.window.showInformationMessage(`Person [${name}] angelegt`);

  // Add to current selection
  const participants = (currentSelection as ParticipantItem[])
    .map(item => item.person)
    .filter(p => p.length > 0);
  participants.push(name);

  // Update the file
  const editor = vscode.window.activeTextEditor;
  if (editor) {
    updateParticipants(editor, participants);
  }
}

/**
 * Update the Participants section in the document
 */
function updateParticipants(editor: vscode.TextEditor, participants: string[]) {
  const content = editor.document.getText();
  const participantsSection = participants.map(p => `- ${p}`).join('\n');

  // Replace or insert participants section
  const participantsRegex = /^### Participants\n([\s\S]*?)(?=\n###|\Z)/m;
  const match = content.match(participantsRegex);

  if (match) {
    // Replace existing
    const newContent = content.replace(
      participantsRegex,
      `### Participants\n${participantsSection}`
    );
    const edit = new vscode.WorkspaceEdit();
    edit.replace(
      editor.document.uri,
      new vscode.Range(0, 0, editor.document.lineCount, 0),
      newContent
    );
    vscode.workspace.applyEdit(edit);
  } else {
    // Insert after first heading
    const headingRegex = /^### \w+/m;
    const headingMatch = content.match(headingRegex);
    if (headingMatch) {
      const insertPos = editor.document.positionAt(content.indexOf(headingMatch[0]));
      const edit = new vscode.WorkspaceEdit();
      edit.insert(
        editor.document.uri,
        insertPos,
        `### Participants\n${participantsSection}\n\n`
      );
      vscode.workspace.applyEdit(edit);
    }
  }
}
