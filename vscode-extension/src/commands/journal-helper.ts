import * as vscode from 'vscode';
import { getFlipClient, NoteResult } from '../flip-client';

/**
 * Configuration for auto-add behavior
 */
const AUTO_ADD_TIMEOUT_SECONDS = 5;

/**
 * Show auto-add to journal popup with countdown
 * Default action is to ADD (if user does nothing)
 * Returns true if added, false if user declined
 */
export async function promptAutoAddToJournal(
  noteData: NoteResult,
  noteType: 'meeting' | 'note' | 'task'
): Promise<boolean> {
  const client = getFlipClient();
  
  // Create a promise that resolves after timeout (auto-add)
  const timeoutPromise = new Promise<'timeout'>((resolve) => {
    setTimeout(() => resolve('timeout'), AUTO_ADD_TIMEOUT_SECONDS * 1000);
  });

  // Show message with countdown hint
  const messagePromise = vscode.window.showInformationMessage(
    `📝 ${noteType === 'meeting' ? 'Meeting' : noteType === 'task' ? 'Task' : 'Note'} erstellt in ${noteData.brain_name}. ` +
    `Wird in ${AUTO_ADD_TIMEOUT_SECONDS}s zum Journal hinzugefügt...`,
    'Jetzt hinzufügen',
    'Nicht ins Journal'
  );

  // Race between user action and timeout
  const result = await Promise.race([messagePromise, timeoutPromise]);

  if (result === 'Nicht ins Journal') {
    // User explicitly declined - add no-journal tag
    await addNoJournalTag(noteData.path);
    vscode.window.showInformationMessage('$(x) Nicht zum Journal hinzugefügt (no-journal Tag gesetzt)');
    return false;
  }

  // Either timeout or "Jetzt hinzufügen" - add to journal
  const linkRes = await client.addToJournal({
    file: noteData.path,
    title: noteData.title,
    type: noteType,
    brain: noteData.brain_name,
  });

  if (!linkRes.success) {
    vscode.window.showWarningMessage(`Journal-Link fehlgeschlagen: ${linkRes.error}`);
    return false;
  }

  vscode.window.showInformationMessage('✓ Im Journal verlinkt');
  return true;
}

/**
 * Add no-journal tag to a file's frontmatter/properties
 */
async function addNoJournalTag(filePath: string): Promise<void> {
  try {
    const doc = await vscode.workspace.openTextDocument(filePath);
    const content = doc.getText();
    
    // Check if file has tags field
    const tagsMatch = content.match(/^- tags:: (.*)$/m);
    
    let newContent: string;
    if (tagsMatch) {
      // Add to existing tags
      const existingTags = tagsMatch[1].trim();
      const newTags = existingTags ? `${existingTags}, no-journal` : 'no-journal';
      newContent = content.replace(/^- tags:: (.*)$/m, `- tags:: ${newTags}`);
    } else {
      // Add tags field after other properties (look for first "- " line that's a property)
      const propertyMatch = content.match(/^(- \w+:: .*)$/m);
      if (propertyMatch) {
        const insertPos = content.indexOf(propertyMatch[0]) + propertyMatch[0].length;
        newContent = content.slice(0, insertPos) + '\n- tags:: no-journal' + content.slice(insertPos);
      } else {
        // Fallback: add at beginning
        newContent = '- tags:: no-journal\n' + content;
      }
    }

    // Apply edit
    const edit = new vscode.WorkspaceEdit();
    edit.replace(
      doc.uri,
      new vscode.Range(0, 0, doc.lineCount, 0),
      newContent
    );
    await vscode.workspace.applyEdit(edit);
    await doc.save();
  } catch (error) {
    console.error('Failed to add no-journal tag:', error);
  }
}

/**
 * Check if a file has the no-journal tag
 */
export function hasNoJournalTag(content: string): boolean {
  const tagsMatch = content.match(/^- tags:: (.*)$/m);
  if (tagsMatch) {
    const tags = tagsMatch[1].toLowerCase().split(',').map(t => t.trim());
    return tags.includes('no-journal');
  }
  return false;
}
