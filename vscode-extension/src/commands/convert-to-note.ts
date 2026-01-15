import * as vscode from 'vscode';
import { getFlipClient, BrainInfo } from '../flip-client';
import * as path from 'path';

// Note types for conversion
const NOTE_TYPES = [
  { label: '$(note) Note', value: 'note', description: 'Regular note' },
  { label: '$(calendar) Meeting Note', value: 'meeting', description: 'Meeting note with participants' },
  { label: '$(heart) Exercise', value: 'exercise', description: 'Exercise definition' }
];

/**
 * Convert the current file to a flip note
 */
export async function convertToFlipNote(): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showErrorMessage('No active editor');
    return;
  }

  const document = editor.document;
  
  // Check if it's a markdown file
  if (document.languageId !== 'markdown' && !document.fileName.endsWith('.md')) {
    const confirm = await vscode.window.showWarningMessage(
      'This is not a markdown file. Convert anyway?',
      'Yes', 'No'
    );
    if (confirm !== 'Yes') {
      return;
    }
  }

  const client = getFlipClient();

  // Get available brains
  const info = await client.getInfo();
  if (!info.success || !info.data) {
    vscode.window.showErrorMessage('Failed to get brain info');
    return;
  }

  // Let user choose brain
  let selectedBrain: BrainInfo | undefined;
  
  if (info.data.brains.length > 1) {
    const brainItems = info.data.brains.map(b => ({
      label: b.active ? `$(star) ${b.name}` : `$(folder) ${b.name}`,
      description: b.active ? 'active' : '',
      detail: `${b.type} - ${b.path}`,
      brain: b
    }));
    
    const selected = await vscode.window.showQuickPick(brainItems, {
      placeHolder: 'Select target brain',
      title: 'Convert to Flip Note'
    });
    
    if (!selected) {
      return;
    }
    selectedBrain = selected.brain;
  } else if (info.data.brains.length === 1) {
    selectedBrain = info.data.brains[0];
  } else {
    vscode.window.showErrorMessage('No brains found');
    return;
  }

  // Let user choose note type
  const noteType = await vscode.window.showQuickPick(NOTE_TYPES, {
    placeHolder: 'Select note type',
    title: 'Convert to Flip Note'
  });

  if (!noteType) {
    return;
  }

  // Analyze current document
  const text = document.getText();
  const existingFrontmatter = parseFrontmatter(text);
  
  // Determine what metadata needs to be added
  const missingFields = getMissingFields(existingFrontmatter, noteType.value);
  
  // Collect missing metadata from user
  const newMetadata = await collectMissingMetadata(missingFields, noteType.value, document.fileName);
  
  if (newMetadata === null) {
    return; // User cancelled
  }

  // Merge existing and new metadata
  const finalMetadata = { ...existingFrontmatter, ...newMetadata };
  
  // Generate the updated content
  const updatedContent = generateUpdatedContent(text, finalMetadata, noteType.value);

  // Apply the changes
  const fullRange = new vscode.Range(
    document.positionAt(0),
    document.positionAt(text.length)
  );

  await editor.edit(editBuilder => {
    editBuilder.replace(fullRange, updatedContent);
  });

  // Save the document
  await document.save();

  vscode.window.showInformationMessage(
    `Converted to ${noteType.label} in ${selectedBrain.name}`
  );

  // Optionally offer to move the file to the brain
  if (!document.fileName.includes(selectedBrain.path)) {
    const moveFile = await vscode.window.showQuickPick([
      { label: '$(file-symlink-file) Move to brain', value: 'move' },
      { label: '$(check) Keep in current location', value: 'keep' }
    ], {
      placeHolder: 'Move file to brain directory?'
    });

    if (moveFile?.value === 'move') {
      await moveFileToBrain(document.fileName, selectedBrain, noteType.value);
    }
  }
}

/**
 * Parse existing frontmatter from document
 */
function parseFrontmatter(text: string): Record<string, any> {
  const frontmatterMatch = text.match(/^---\n([\s\S]*?)\n---/);
  if (!frontmatterMatch) {
    return {};
  }

  const frontmatter: Record<string, any> = {};
  const lines = frontmatterMatch[1].split('\n');
  
  for (const line of lines) {
    const match = line.match(/^(\w+):\s*(.*)$/);
    if (match) {
      const [, key, value] = match;
      // Handle arrays like tags: [tag1, tag2]
      if (value.startsWith('[') && value.endsWith(']')) {
        frontmatter[key] = value.slice(1, -1).split(',').map(s => s.trim()).filter(Boolean);
      } else {
        frontmatter[key] = value.trim();
      }
    }
  }

  return frontmatter;
}

/**
 * Determine which fields are missing for the note type
 */
function getMissingFields(existing: Record<string, any>, noteType: string): string[] {
  const missing: string[] = [];
  
  // Common required fields
  if (!existing.title) missing.push('title');
  if (!existing.created) missing.push('created');
  
  // Type-specific fields
  switch (noteType) {
    case 'meeting':
      if (!existing.participants) missing.push('participants');
      if (!existing.date) missing.push('date');
      break;
    case 'exercise':
      if (!existing.id) missing.push('id');
      if (!existing.context) missing.push('context');
      break;
    case 'note':
      if (!existing.tags) missing.push('tags');
      break;
  }

  return missing;
}

/**
 * Collect missing metadata from user
 */
async function collectMissingMetadata(
  missingFields: string[],
  noteType: string,
  fileName: string
): Promise<Record<string, any> | null> {
  const metadata: Record<string, any> = {};
  const today = new Date().toISOString().split('T')[0];
  const baseName = path.basename(fileName, '.md');

  for (const field of missingFields) {
    switch (field) {
      case 'title': {
        const title = await vscode.window.showInputBox({
          prompt: 'Enter note title',
          value: baseName.replace(/-/g, ' ').replace(/_/g, ' '),
          placeHolder: 'Note title'
        });
        if (title === undefined) return null;
        metadata.title = title || baseName;
        break;
      }

      case 'created': {
        metadata.created = today;
        break;
      }

      case 'date': {
        const date = await vscode.window.showInputBox({
          prompt: 'Enter meeting date (YYYY-MM-DD)',
          value: today,
          placeHolder: 'YYYY-MM-DD'
        });
        if (date === undefined) return null;
        metadata.date = date || today;
        break;
      }

      case 'participants': {
        const participants = await vscode.window.showInputBox({
          prompt: 'Enter participants (comma-separated)',
          placeHolder: 'John Doe, Jane Smith'
        });
        if (participants === undefined) return null;
        metadata.participants = participants ? 
          participants.split(',').map(p => p.trim()) : [];
        break;
      }

      case 'tags': {
        const tags = await vscode.window.showInputBox({
          prompt: 'Enter tags (comma-separated)',
          placeHolder: 'tag1, tag2'
        });
        if (tags === undefined) return null;
        metadata.tags = tags ? tags.split(',').map(t => t.trim()) : [];
        break;
      }

      case 'id': {
        // Generate a unique ID for exercises
        const id = baseName.toLowerCase()
          .replace(/[^a-z0-9]+/g, '-')
          .replace(/^-|-$/g, '');
        metadata.id = id;
        break;
      }

      case 'context': {
        const context = await vscode.window.showInputBox({
          prompt: 'Enter exercise context (e.g., gym, home, office)',
          placeHolder: 'Context'
        });
        if (context === undefined) return null;
        metadata.context = context || 'general';
        break;
      }
    }
  }

  return metadata;
}

/**
 * Generate updated document content with proper frontmatter
 */
function generateUpdatedContent(
  originalText: string,
  metadata: Record<string, any>,
  noteType: string
): string {
  // Remove existing frontmatter if present
  let content = originalText.replace(/^---\n[\s\S]*?\n---\n*/, '');
  
  // Build new frontmatter
  const frontmatterLines: string[] = ['---'];
  
  // Add type-specific metadata in a logical order
  const fieldOrder = noteType === 'exercise'
    ? ['id', 'title', 'context', 'created', 'tags']
    : noteType === 'meeting'
    ? ['title', 'date', 'participants', 'created', 'tags']
    : ['title', 'created', 'tags'];

  for (const field of fieldOrder) {
    if (metadata[field] !== undefined) {
      if (Array.isArray(metadata[field])) {
        frontmatterLines.push(`${field}: [${metadata[field].join(', ')}]`);
      } else {
        frontmatterLines.push(`${field}: ${metadata[field]}`);
      }
    }
  }

  // Add any remaining fields not in the order list
  for (const [key, value] of Object.entries(metadata)) {
    if (!fieldOrder.includes(key)) {
      if (Array.isArray(value)) {
        frontmatterLines.push(`${key}: [${value.join(', ')}]`);
      } else {
        frontmatterLines.push(`${key}: ${value}`);
      }
    }
  }

  frontmatterLines.push('---');
  frontmatterLines.push('');

  // Ensure content has a title heading if not present
  if (!content.trim().startsWith('#') && metadata.title) {
    content = `# ${metadata.title}\n\n${content.trim()}`;
  }

  return frontmatterLines.join('\n') + content;
}

/**
 * Move file to the appropriate location in the brain
 */
async function moveFileToBrain(
  currentPath: string,
  brain: BrainInfo,
  noteType: string
): Promise<void> {
  const fileName = path.basename(currentPath);
  
  // Determine target directory based on note type
  let targetDir: string;
  switch (noteType) {
    case 'meeting':
      targetDir = path.join(brain.path, 'meetings');
      break;
    case 'exercise':
      targetDir = path.join(brain.path, 'exercises');
      break;
    default:
      targetDir = path.join(brain.path, 'notes');
  }

  const targetPath = path.join(targetDir, fileName);

  try {
    // Create directory if needed
    await vscode.workspace.fs.createDirectory(vscode.Uri.file(targetDir));
    
    // Move the file
    await vscode.workspace.fs.rename(
      vscode.Uri.file(currentPath),
      vscode.Uri.file(targetPath),
      { overwrite: false }
    );

    // Open the moved file
    const doc = await vscode.workspace.openTextDocument(targetPath);
    await vscode.window.showTextDocument(doc);

    vscode.window.showInformationMessage(`Moved to ${targetPath}`);
  } catch (error: any) {
    vscode.window.showErrorMessage(`Failed to move file: ${error.message}`);
  }
}
