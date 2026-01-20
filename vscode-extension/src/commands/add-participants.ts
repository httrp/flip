import * as vscode from 'vscode';
import { getFlipClient, FlipResult, DefinitionItem } from '../flip-client';

/**
 * Participant with optional abbreviation for @-mentions
 */
interface Participant {
  name: string;
  abbreviation?: string;
}

/**
 * Participant item for quick pick
 */
interface ParticipantItem extends vscode.QuickPickItem {
  person: string;
  abbreviation?: string;
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

  // Get the current participants if any (parse name and potential abbreviation)
  let currentParticipants: Participant[] = [];
  // Match Participants section regardless of # count (## or ### or even #)
  const participantsMatch = content.match(/^#+\s+Participants\s*\n([\s\S]*?)(?=\n#+\s+|\Z)/m);
  if (participantsMatch) {
    const participantsText = participantsMatch[1].trim();
    if (participantsText && participantsText !== '-') {
      currentParticipants = participantsText
        .split('\n')
        .map(line => {
          const text = line.replace(/^-\s*/, '').trim();
          // Check for pattern: "Name (@abbrev)"
          const abbrevMatch = text.match(/^(.+?)\s+\(@(\w+)\)$/);
          if (abbrevMatch) {
            return { name: abbrevMatch[1].trim(), abbreviation: abbrevMatch[2] };
          }
          return { name: text, abbreviation: undefined };
        })
        .filter(p => p.name.length > 0);
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

    // Helper to check if participant is already selected
    const isParticipantSelected = (name: string): boolean => {
      return currentParticipants.some(p => p.name === name || p.name.startsWith(name + ' ('));
    };

    // Helper to find abbreviation for existing participant
    const findAbbreviation = (name: string): string | undefined => {
      const found = currentParticipants.find(p => p.name === name || p.name.startsWith(name + ' ('));
      return found?.abbreviation;
    };

    // Add historical series participants first (for series meetings)
    if (seriesParticipants.length > 0) {
      items.push({
        label: '── Aus Serie (historisch) ──',
        kind: vscode.QuickPickItemKind.Separator,
        person: '',
      });
      for (const p of seriesParticipants) {
        seenParticipants.add(p);
        const isSelected = isParticipantSelected(p);
        items.push({
          label: `${isSelected ? '$(check)' : '$(circle-outline)'} ${p}`,
          person: p,
          abbreviation: findAbbreviation(p),
          isFromSeries: true,
          picked: isSelected,
          description: isSelected ? 'ausgewählt' : '',
        });
      }
    }

    // Add people from definitions (with their abbreviations)
    if (defsResult.data.people.length > 0) {
      items.push({
        label: '── Definitionen ──',
        kind: vscode.QuickPickItemKind.Separator,
        person: '',
      });
      for (const person of defsResult.data.people) {
        const display = person.organization ? `${person.name} (${person.organization})` : person.name;
        if (!seenParticipants.has(display)) {
          const isSelected = isParticipantSelected(person.name);
          // Show abbreviation in description
          const abbrevDesc = person.abbreviation ? `@${person.abbreviation}` : '';
          items.push({
            label: `${isSelected ? '$(check)' : '$(circle-outline)'} ${display}`,
            person: display,
            abbreviation: person.abbreviation,
            picked: isSelected,
            description: isSelected ? `ausgewählt ${abbrevDesc}`.trim() : abbrevDesc,
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
      // Get selected participants with abbreviations
      const selectedParticipants: Participant[] = withoutNew
        .filter(item => item.person.length > 0)
        .map(item => ({ name: item.person, abbreviation: item.abbreviation }));

      // Add new person
      const newPerson = await addNewPersonInteractive(client, brainName);
      if (newPerson) {
        // Automatically add the new person to selection
        const alreadyExists = currentParticipants.some(p => p.name === newPerson.name);
        if (!alreadyExists) {
          currentParticipants.push(newPerson);
        }
        // Continue loop to show picker again
        content = editor.document.getText(); // Refresh content
      }
      // Continue loop
    } else if (isDone) {
      // Save and exit
      done = true;
      // Update participants in the file with current selection (with abbreviations)
      const finalSelected: Participant[] = selected
        .filter(item => item.person !== '__done__' && item.person.length > 0)
        .map(item => ({ name: item.person, abbreviation: item.abbreviation }));

      // Merge with existing, keeping abbreviations
      const mergedMap = new Map<string, Participant>();
      for (const p of currentParticipants) {
        mergedMap.set(p.name, p);
      }
      for (const p of finalSelected) {
        if (!mergedMap.has(p.name)) {
          mergedMap.set(p.name, p);
        }
      }
      const allParticipants = Array.from(mergedMap.values());

      if (allParticipants.length === 0) {
        vscode.window.showWarningMessage('Keine Teilnehmer ausgewählt');
        return;
      }

      const success = await updateParticipants(editor, allParticipants);
      if (success) {
        vscode.window.showInformationMessage(`${allParticipants.length} Teilnehmer gespeichert`);
      }
    } else {
      // Update selection (without __new__ or __done__) - with abbreviations
      currentParticipants = selected
        .filter(item => item.person !== '__new__' && item.person !== '__done__' && item.person.length > 0)
        .map(item => ({ name: item.person, abbreviation: item.abbreviation }));

      if (currentParticipants.length === 0) {
        vscode.window.showWarningMessage('Keine Teilnehmer ausgewählt');
        return;
      }

      done = true;
      const success = await updateParticipants(editor, currentParticipants);
      if (success) {
        vscode.window.showInformationMessage(`${currentParticipants.length} Teilnehmer hinzugefügt`);
      }
    }
  }
}

/**
 * Handle adding a new person interactively
 */
async function addNewPersonInteractive(
  client: ReturnType<typeof getFlipClient>,
  brainName?: string
): Promise<{ name: string; abbreviation?: string } | undefined> {
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

  // Prompt for abbreviation/Kürzel
  const defaultAbbrev = generateAbbreviation(name.trim());
  const abbreviation = await vscode.window.showInputBox({
    prompt: 'Kürzel für @-Mentions (optional)',
    placeHolder: `z.B. ${defaultAbbrev}`,
    value: defaultAbbrev,
  });

  // Get organizations for picker
  const defsResult = await client.listDefinitions(brainName);
  let selectedOrg: string | undefined;
  let isNewOrg = false;

  if (defsResult.success && defsResult.data) {
    // Build org picker items
    interface OrgPickItem extends vscode.QuickPickItem {
      orgName: string;
      isNew?: boolean;
    }

    const orgItems: OrgPickItem[] = defsResult.data.organizations.map(org => ({
      label: org.name,
      description: org.abbreviation ? `@${org.abbreviation}` : '',
      orgName: org.name,
    }));

    // Add "new org" option
    orgItems.push({
      label: '$(plus) Neue Organisation anlegen',
      orgName: '__new__',
      isNew: true,
    });

    // Add "keine" option at the beginning
    orgItems.unshift({
      label: '$(circle-slash) Keine Organisation',
      orgName: '',
    });

    const orgChoice = await vscode.window.showQuickPick(orgItems, {
      placeHolder: 'Organisation auswählen (optional)',
    });

    if (orgChoice) {
      if (orgChoice.orgName === '__new__') {
        // Create new organization
        const newOrgName = await vscode.window.showInputBox({
          prompt: 'Name der neuen Organisation',
          placeHolder: 'z.B. Acme Corp',
        });

        if (newOrgName) {
          const newOrgAbbrev = await vscode.window.showInputBox({
            prompt: 'Kürzel für die Organisation',
            placeHolder: 'z.B. ACME',
            value: generateOrgAbbreviation(newOrgName),
          });

          // Save new organization
          const orgResult = await client.addOrganization({
            abbreviation: newOrgAbbrev || generateOrgAbbreviation(newOrgName),
            name: newOrgName,
            brain: brainName,
          });

          if (orgResult.success) {
            vscode.window.showInformationMessage(`✓ Organisation angelegt: ${newOrgName}`);
            selectedOrg = newOrgName;
            isNewOrg = true;
          } else {
            vscode.window.showWarningMessage(`Organisation konnte nicht gespeichert werden: ${orgResult.error}`);
            selectedOrg = newOrgName; // Still use it for the person
          }
        }
      } else if (orgChoice.orgName) {
        selectedOrg = orgChoice.orgName;
      }
    }
  } else {
    // Fallback: just ask for org name as text
    selectedOrg = await vscode.window.showInputBox({
      prompt: 'Organisation (optional)',
      placeHolder: 'z.B. Acme Corp',
    }) || undefined;
  }

  // Prompt for role
  const role = await vscode.window.showInputBox({
    prompt: 'Rolle (optional)',
    placeHolder: 'z.B. Project Manager',
  });

  // Add to definitions (with brain context)
  const result = await client.addPerson({
    name: name.trim(),
    abbreviation: abbreviation?.trim() || undefined,
    organization: selectedOrg?.trim() || undefined,
    role: role?.trim() || undefined,
    brain: brainName,
  });

  if (!result.success) {
    vscode.window.showErrorMessage(`Fehler beim Anlegen: ${result.error}`);
    return undefined;
  }

  vscode.window.showInformationMessage(`✓ Person angelegt: ${name}${abbreviation ? ` (@${abbreviation})` : ''}`);
  return { name: name.trim(), abbreviation: abbreviation?.trim() };
}

/**
 * Generate a default abbreviation from a name (first letters of words, uppercase)
 */
function generateAbbreviation(name: string): string {
  return name
    .split(/\s+/)
    .map(word => word.charAt(0).toLowerCase())
    .join('');
}

/**
 * Generate a default abbreviation for an organization (uppercase, max 4 chars)
 */
function generateOrgAbbreviation(name: string): string {
  const words = name.split(/\s+/);
  if (words.length === 1) {
    return name.substring(0, 4).toUpperCase();
  }
  return words
    .map(word => word.charAt(0).toUpperCase())
    .join('')
    .substring(0, 4);
}

/**
 * Update the Participants section in the document
 */
async function updateParticipants(editor: vscode.TextEditor, participants: Participant[]): Promise<boolean> {
  const content = editor.document.getText();
  // Format participants with optional abbreviation: "- Name (@abbrev)" or just "- Name"
  const participantsSection = participants.map(p => {
    if (p.abbreviation) {
      return `- ${p.name} (@${p.abbreviation})`;
    }
    return `- ${p.name}`;
  }).join('\n');

  // Try to find existing Participants section (any # level)
  const participantsRegex = /^(#+\s+Participants\s*)\n([\s\S]*?)(?=\n#+\s+|\Z)/m;
  const match = content.match(participantsRegex);

  const edit = new vscode.WorkspaceEdit();

  if (match) {
    // Replace existing participants section, keeping the original format
    const start = content.indexOf(match[0]);
    const end = start + match[0].length;
    const startPos = editor.document.positionAt(start);
    const endPos = editor.document.positionAt(end);
    const range = new vscode.Range(startPos, endPos);
    
    const newSection = `${match[1]}\n${participantsSection}`;
    edit.replace(editor.document.uri, range, newSection);
    console.log(`[flip] Replacing participants section at ${start}-${end}`);
  } else {
    // Section doesn't exist - just insert it after the first heading or at a good spot
    // Try to insert after frontmatter (---)
    const frontmatterEnd = content.indexOf('\n---\n');
    let insertPos = 0;
    
    if (frontmatterEnd !== -1) {
      // Found frontmatter, insert after it
      insertPos = frontmatterEnd + 5; // After the second ---\n
    } else {
      // No frontmatter, try to insert after first heading
      const firstHeadingMatch = content.match(/^#+\s+\w+.*?$/m);
      if (firstHeadingMatch) {
        const headingStart = content.indexOf(firstHeadingMatch[0]);
        const headingEnd = headingStart + firstHeadingMatch[0].length;
        insertPos = content.indexOf('\n', headingEnd) + 1; // After heading line
      }
    }

    const pos = editor.document.positionAt(insertPos);
    edit.insert(editor.document.uri, pos, `\n## Participants\n${participantsSection}\n`);
    console.log(`[flip] Inserting new participants section at position ${insertPos}`);
  }

  // IMPORTANT: await the edit and check if it succeeded
  const success = await vscode.workspace.applyEdit(edit);
  if (!success) {
    console.error('[flip] Failed to apply edit to document');
    vscode.window.showErrorMessage('Fehler beim Speichern der Teilnehmer');
  }
  return success;
}
