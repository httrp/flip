import * as vscode from 'vscode';
import { getFlipClient, FlipResult, DefinitionItem } from '../flip-client';

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
export async function addParticipantsCommand() {
  const client = getFlipClient();
  // Get active editor
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showErrorMessage('Bitte öffne zuerst eine Datei');
    return;
  }

  const filePath = editor.document.fileName;
  let content = editor.document.getText();

  // Determine brain from file path
  const brainName = await client.getBrainForPath(filePath);

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

  // Interactive loop for adding participants
  let done = false;
  while (!done) {
    // Get all definitions (from the same brain as the file)
    const defsResult = await client.listDefinitions(brainName);
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
      description: 'Und direkt zum Meeting hinzufügen',
    });
    items.push({
      label: '$(check-all) Fertig - Speichern',
      person: '__done__',
      description: `${currentParticipants.length} Teilnehmer ausgewählt`,
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
    const isDone = selected.some(item => item.person === '__done__');

    if (hasNewPerson) {
      // Remove __new__ from selection
      const withoutNew = selected.filter(item => item.person !== '__new__' && item.person !== '__done__');
      // Get selected participants
      const selectedParticipants = withoutNew
        .map(item => item.person)
        .filter(p => p.length > 0);

      // Add new person
      const newPersonName = await addNewPersonInteractive(client, brainName);
      if (newPersonName) {
        // Automatically add the new person to selection
        if (!currentParticipants.includes(newPersonName)) {
          currentParticipants.push(newPersonName);
        }
        // Continue loop to show picker again
        content = editor.document.getText(); // Refresh content
      }
      // Continue loop
    } else if (isDone) {
      // Save and exit
      done = true;
      // Update participants in the file with current selection
      const finalSelected = selected
        .filter(item => item.person !== '__done__')
        .map(item => item.person)
        .filter(p => p.length > 0);

      // Merge with existing
      const allParticipants = Array.from(new Set([...currentParticipants, ...finalSelected]));

      if (allParticipants.length === 0) {
        vscode.window.showWarningMessage('Keine Teilnehmer ausgewählt');
        return;
      }

      updateParticipants(editor, allParticipants);
      vscode.window.showInformationMessage(`${allParticipants.length} Teilnehmer gespeichert`);
    } else {
      // Update selection (without __new__ or __done__)
      currentParticipants = selected
        .filter(item => item.person !== '__new__' && item.person !== '__done__')
        .map(item => item.person)
        .filter(p => p.length > 0);

      if (currentParticipants.length === 0) {
        vscode.window.showWarningMessage('Keine Teilnehmer ausgewählt');
        return;
      }

      done = true;
      updateParticipants(editor, currentParticipants);
      vscode.window.showInformationMessage(`${currentParticipants.length} Teilnehmer hinzugefügt`);
    }
  }
}

/**
 * Handle adding a new person interactively
 */
async function addNewPersonInteractive(
  client: ReturnType<typeof getFlipClient>,
  brainName?: string
): Promise<string | undefined> {
  // Prompt for name
  const name = await vscode.window.showInputBox({
    prompt: 'Name der neuen Person',
    placeHolder: 'z.B. John Doe',
    validateInput: (value) => {
      if (!value || value.trim() === '') {
        return 'Name ist erforderlich';
      }
      return null;
    },
  });

  if (!name) {
    return undefined;
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

  // Add to definitions (with brain context)
  const result = await client.addPerson({
    name: name.trim(),
    organization: org?.trim() || undefined,
    role: role?.trim() || undefined,
    brain: brainName,
  });

  if (!result.success) {
    vscode.window.showErrorMessage(`Fehler beim Anlegen: ${result.error}`);
    return undefined;
  }

  vscode.window.showInformationMessage(`✓ Person angelegt: ${name}`);
  return name.trim();
}

/**
 * Update the Participants section in the document
 */
function updateParticipants(editor: vscode.TextEditor, participants: string[]) {
  const content = editor.document.getText();
  const participantsSection = participants.map(p => `- ${p}`).join('\n');

  // Check if participants section exists
  const participantsRegex = /^### Participants\n([\s\S]*?)(?=\n###|\Z)/m;
  const match = content.match(participantsRegex);

  const edit = new vscode.WorkspaceEdit();

  if (match) {
    // Replace existing participants section
    const start = content.indexOf(match[0]);
    const end = start + match[0].length;
    const startPos = editor.document.positionAt(start);
    const endPos = editor.document.positionAt(end);
    const range = new vscode.Range(startPos, endPos);
    
    const newSection = `### Participants\n${participantsSection}`;
    edit.replace(editor.document.uri, range, newSection);
  } else {
    // Find first ### heading to insert after
    const firstHeadingRegex = /^(###\s+\w+.*?)$/m;
    const headingMatch = content.match(firstHeadingRegex);
    
    if (headingMatch) {
      const insertPos = content.indexOf(headingMatch[0]) + headingMatch[0].length;
      const pos = editor.document.positionAt(insertPos);
      edit.insert(editor.document.uri, pos, `\n\n### Participants\n${participantsSection}`);
    } else {
      // Fallback: insert at beginning
      const pos = editor.document.positionAt(0);
      edit.insert(editor.document.uri, pos, `### Participants\n${participantsSection}\n\n`);
    }
  }

  vscode.workspace.applyEdit(edit);
}
