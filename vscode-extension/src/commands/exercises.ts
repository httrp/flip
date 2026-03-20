import * as vscode from 'vscode';
import { getFlipClient, ExerciseItemResult, ExercisesListResult, ExerciseDetailResult, ExerciseTrackResult, ExerciseNewResult } from '../flip-client';

let exerciseOutput: vscode.OutputChannel | undefined;
function getExerciseOutput(): vscode.OutputChannel {
  if (!exerciseOutput) { exerciseOutput = vscode.window.createOutputChannel('Flip Exercise'); }
  return exerciseOutput;
}

interface ExerciseQuickPickItem extends vscode.QuickPickItem {
  exercise: ExerciseItemResult;
}

/**
 * Main exercise command - shows list and allows actions
 */
export async function exerciseCommand(): Promise<void> {
  const client = getFlipClient();

  // Get exercises
  const result = await client.listExercises();

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(result.error || 'Failed to load exercises');
    return;
  }

  const exercises = result.data.exercises;

  if (exercises.length === 0) {
    const action = await vscode.window.showInformationMessage(
      'Keine Exercises gefunden. Möchtest du ein neues erstellen?',
      'Ja', 'Nein'
    );
    if (action === 'Ja') {
      await newExerciseCommand();
    }
    return;
  }

  // Build quick pick items
  const items: ExerciseQuickPickItem[] = exercises.map(ex => {
    const contextStr = ex.context ? `[${ex.context}]` : '';
    const sessionStr = ex.session_count > 0 ? `${ex.session_count} Sessions` : 'Noch nicht getracked';
    const lastStr = ex.last_session ? `, zuletzt: ${ex.last_session}` : '';

    return {
      label: `🏋️ ${ex.name}`,
      description: contextStr,
      detail: `${sessionStr}${lastStr} • ${ex.variant_count} Varianten`,
      exercise: ex,
    };
  });

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Exercise auswählen',
    matchOnDescription: true,
    matchOnDetail: true,
  });

  if (!selected) {
    return;
  }

  // Show actions for selected exercise
  const action = await vscode.window.showQuickPick([
    { label: '📝 Tracken', value: 'track' },
    { label: '👁️ Anzeigen', value: 'view' },
    { label: '📂 Datei öffnen', value: 'open' },
  ], {
    placeHolder: `Aktion für "${selected.exercise.name}"`,
  });

  if (!action) {
    return;
  }

  switch (action.value) {
    case 'track':
      await trackExerciseSession(client, selected.exercise);
      break;
    case 'view':
      await showExerciseDetails(client, selected.exercise);
      break;
    case 'open':
      const doc = await vscode.workspace.openTextDocument(selected.exercise.file_path);
      await vscode.window.showTextDocument(doc);
      break;
  }
}

/**
 * Track an exercise session
 */
async function trackExerciseSession(client: ReturnType<typeof getFlipClient>, exercise: ExerciseItemResult): Promise<void> {
  // Ask for duration
  const durationStr = await vscode.window.showInputBox({
    prompt: 'Dauer in Minuten (optional)',
    placeHolder: 'z.B. 30',
    validateInput: (value) => {
      if (value && isNaN(parseInt(value))) {
        return 'Bitte eine Zahl eingeben';
      }
      return null;
    },
  });

  if (durationStr === undefined) {
    return; // Cancelled
  }

  const duration = durationStr ? parseInt(durationStr) : 0;

  // Ask for notes
  const notes = await vscode.window.showInputBox({
    prompt: 'Notizen (optional)',
    placeHolder: 'z.B. Gute Session, Fokus auf Technik',
  });

  if (notes === undefined) {
    return; // Cancelled
  }

  // Track the session
  const result = await client.trackExercise({
    exerciseId: exercise.id,
    duration,
    notes,
  });

  if (result.success && result.data) {
    vscode.window.showInformationMessage(
      `✅ Session für "${exercise.name}" getrackt!`
    );

    // Ask to open journal
    const openJournal = await vscode.window.showInformationMessage(
      'Journal öffnen?',
      'Ja', 'Nein'
    );

    if (openJournal === 'Ja') {
      const doc = await vscode.workspace.openTextDocument(result.data.journal_path);
      await vscode.window.showTextDocument(doc);
    }
  } else {
    vscode.window.showErrorMessage(result.error || 'Tracking fehlgeschlagen');
  }
}

/**
 * Show exercise details
 */
async function showExerciseDetails(client: ReturnType<typeof getFlipClient>, exercise: ExerciseItemResult): Promise<void> {
  const result = await client.showExercise(exercise.id);

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(result.error || 'Konnte Details nicht laden');
    return;
  }

  const ex = result.data;

  // Build detail message
  let details = `# ${ex.name}\n\n`;
  
  if (ex.context) {
    details += `**Context:** ${ex.context}\n`;
  }
  if (ex.description) {
    details += `**Beschreibung:** ${ex.description}\n`;
  }
  if (ex.goal) {
    details += `**Ziel:** ${ex.goal}\n`;
  }
  
  details += `\n**Sessions:** ${ex.session_count}`;
  if (ex.last_session) {
    details += ` (zuletzt: ${ex.last_session})`;
  }
  details += '\n';

  if (ex.variants && ex.variants.length > 0) {
    details += `\n## Varianten (${ex.variants.length})\n`;
    for (const v of ex.variants) {
      details += `- ${v.name || 'Standard'}`;
      if (v.description) {
        details += `: ${v.description}`;
      }
      details += '\n';
    }
  }

  // Show in output channel or as notification
  const outputChannel = getExerciseOutput();
  outputChannel.clear();
  outputChannel.appendLine(details);
  outputChannel.show();
}

/**
 * Create a new exercise
 */
export async function newExerciseCommand(): Promise<void> {
  const client = getFlipClient();

  // Get name
  const name = await vscode.window.showInputBox({
    prompt: 'Exercise Name',
    placeHolder: 'z.B. Liegestütze, Skalenkontrolle, Vokabeln',
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

  // Get context
  const context = await vscode.window.showInputBox({
    prompt: 'Context (optional)',
    placeHolder: 'z.B. Fitness, Musik, Sprachen',
  });

  if (context === undefined) {
    return;
  }

  // Get description
  const description = await vscode.window.showInputBox({
    prompt: 'Beschreibung (optional)',
    placeHolder: 'Kurze Beschreibung der Übung',
  });

  if (description === undefined) {
    return;
  }

  // Get goal
  const goal = await vscode.window.showInputBox({
    prompt: 'Ziel (optional)',
    placeHolder: 'z.B. 50 Wiederholungen, Geschwindigkeit 120bpm',
  });

  if (goal === undefined) {
    return;
  }

  // Create exercise
  const result = await client.createExercise({
    name,
    context: context || undefined,
    description: description || undefined,
    goal: goal || undefined,
  });

  if (result.success && result.data) {
    const openFile = await vscode.window.showInformationMessage(
      `✅ Exercise "${name}" erstellt!`,
      'Datei öffnen', 'OK'
    );

    if (openFile === 'Datei öffnen') {
      const doc = await vscode.workspace.openTextDocument(result.data.file_path);
      await vscode.window.showTextDocument(doc);
    }
  } else {
    vscode.window.showErrorMessage(result.error || 'Erstellung fehlgeschlagen');
  }
}

/**
 * Quick track - select exercise and track immediately
 */
export async function quickTrackCommand(): Promise<void> {
  const client = getFlipClient();

  // Get exercises
  const result = await client.listExercises();

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(result.error || 'Failed to load exercises');
    return;
  }

  const exercises = result.data.exercises;

  if (exercises.length === 0) {
    vscode.window.showInformationMessage('Keine Exercises zum Tracken gefunden.');
    return;
  }

  // Build quick pick items
  const items: ExerciseQuickPickItem[] = exercises.map(ex => {
    const contextStr = ex.context ? `[${ex.context}]` : '';

    return {
      label: `🏋️ ${ex.name}`,
      description: contextStr,
      exercise: ex,
    };
  });

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Exercise zum Tracken auswählen',
    matchOnDescription: true,
  });

  if (!selected) {
    return;
  }

  await trackExerciseSession(client, selected.exercise);
}
