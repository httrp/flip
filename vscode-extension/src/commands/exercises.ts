import * as vscode from 'vscode';
import { getFlipClient, ExerciseItemResult, ExercisesListResult, ExerciseDetailResult, ExerciseTrackResult, ExerciseNewResult, ExerciseVariantResult } from '../flip-client';

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
    { label: '🧹 Sessions migrieren', value: 'migrate' },
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
    case 'migrate':
      await migrateExerciseSessionsCommand();
      break;
  }
}

export async function migrateExerciseSessionsCommand(): Promise<void> {
  const client = getFlipClient();

  const mode = await vscode.window.showQuickPick([
    { label: 'Nur prüfen (Dry-Run)', value: 'dry-run' },
    { label: 'Migrieren & schreiben', value: 'apply' },
  ], {
    placeHolder: 'Exercise-Sessions migrieren',
  });

  if (!mode) {
    return;
  }

  let days: number | undefined;
  const daysInput = await vscode.window.showInputBox({
    prompt: 'Nur letzte N Tage migrieren (leer = alle)',
    placeHolder: 'z.B. 90',
    validateInput: (value) => {
      if (!value.trim()) {
        return null;
      }
      if (Number.isNaN(Number(value)) || Number(value) < 1) {
        return 'Bitte eine positive Zahl eingeben';
      }
      return null;
    },
  });

  if (daysInput === undefined) {
    return;
  }
  if (daysInput.trim()) {
    days = Number(daysInput.trim());
  }

  const res = await client.migrateExerciseSessions({
    apply: mode.value === 'apply',
    backup: true,
    days,
  });

  if (!res.success || !res.data) {
    vscode.window.showErrorMessage(res.error || 'Migration fehlgeschlagen');
    return;
  }

  const data = res.data;
  const output = getExerciseOutput();
  output.clear();
  output.appendLine('Exercise Session Migration');
  output.appendLine('');
  output.appendLine(`Mode: ${data.dry_run ? 'Dry-Run' : 'Apply'}`);
  output.appendLine(`Files scanned: ${data.files_scanned}`);
  output.appendLine(`Files changed: ${data.files_changed}`);
  output.appendLine(`Sessions migrated: ${data.sessions_migrated}`);
  output.appendLine(`Legacy CLI sessions: ${data.legacy_cli_sessions}`);
  output.appendLine(`Legacy VS Code sessions: ${data.legacy_vscode_sessions}`);
  output.appendLine(`Already compact: ${data.already_compact}`);
  output.appendLine(`Backups created: ${data.backups_created}`);
  if (data.errors && data.errors.length > 0) {
    output.appendLine('');
    output.appendLine('Warnings:');
    for (const e of data.errors) {
      output.appendLine(`- ${e}`);
    }
  }
  output.show();

  const suffix = data.dry_run ? ' (Dry-Run)' : '';
  vscode.window.showInformationMessage(`✅ Exercise-Migration abgeschlossen${suffix}: ${data.sessions_migrated} Session(s) verarbeitet`);
}

/**
 * Track an exercise session
 */
async function trackExerciseSession(client: ReturnType<typeof getFlipClient>, exercise: ExerciseItemResult): Promise<void> {
  let selectedVariant: string | undefined;
  const trackedProperties: Record<string, string | number> = {};

  const detailsResult = await client.showExercise(exercise.id);
  if (detailsResult.success && detailsResult.data?.variants && detailsResult.data.variants.length > 0) {
    const variants = detailsResult.data.variants;
    if (variants.length > 1) {
      const variantItems = variants.map((v) => ({
        label: v.name || 'Standard',
        description: v.description || '',
        value: v,
      }));
      const selected = await vscode.window.showQuickPick(variantItems, {
        placeHolder: 'Variante auswählen (optional)',
        matchOnDescription: true,
      });

      if (selected === undefined) {
        return;
      }
      selectedVariant = selected.value.name;

      await promptTrackingProperties(selected.value.tracking_properties || {}, trackedProperties);
    } else {
      const onlyVariant = variants[0];
      selectedVariant = onlyVariant.name;
      await promptTrackingProperties(onlyVariant.tracking_properties || {}, trackedProperties);
    }
  }

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
    variant: selectedVariant,
    notes,
    properties: trackedProperties,
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

async function promptTrackingProperties(
  trackingProperties: Record<string, string>,
  target: Record<string, string | number>
): Promise<void> {
  const propertyEntries = Object.entries(trackingProperties);
  for (const [name, unit] of propertyEntries) {
    if (name === 'duration_min') {
      continue;
    }

    const input = await vscode.window.showInputBox({
      prompt: `Wert für ${name} (${unit})`,
      placeHolder: 'Optional',
    });

    if (input === undefined) {
      continue;
    }

    const trimmed = input.trim();
    if (!trimmed) {
      continue;
    }

    const asNumber = Number(trimmed);
    target[name] = Number.isNaN(asNumber) ? trimmed : asNumber;
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
    placeHolder: 'z.B. Ausdauer verbessern',
  });

  if (goal === undefined) {
    return;
  }

  const variants = await promptVariantsForNewExercise();
  if (variants === undefined) {
    return;
  }

  // Create exercise
  const result = await client.createExercise({
    name,
    context: context || undefined,
    description: description || undefined,
    goal: goal || undefined,
    variants,
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

async function promptVariantsForNewExercise(): Promise<ExerciseVariantResult[] | undefined> {
  const addVariants = await vscode.window.showQuickPick(
    [
      { label: 'Ja', value: 'yes' },
      { label: 'Nein', value: 'no' },
    ],
    {
      placeHolder: 'Möchtest du direkt Varianten für diese Übung anlegen?',
    }
  );

  if (!addVariants) {
    return undefined;
  }

  if (addVariants.value === 'no') {
    return [];
  }

  const variants: ExerciseVariantResult[] = [];

  while (true) {
    const variantName = await vscode.window.showInputBox({
      prompt: `Variant-Name (optional)${variants.length > 0 ? ` #${variants.length + 1}` : ''}`,
      placeHolder: 'z.B. Normales Laufen, Intervalllauf, Technikfokus',
    });
    if (variantName === undefined) {
      return undefined;
    }

    const variantDescription = await vscode.window.showInputBox({
      prompt: 'Variant-Beschreibung (optional)',
      placeHolder: 'Was ist an dieser Variante speziell?',
    });
    if (variantDescription === undefined) {
      return undefined;
    }

    const trackingProperties = await promptTrackingPropertiesForVariant();
    if (trackingProperties === undefined) {
      return undefined;
    }

    const hasVariantContent =
      variantName.trim() !== '' ||
      variantDescription.trim() !== '' ||
      Object.keys(trackingProperties).length > 0;

    if (!hasVariantContent) {
      const emptyAction = await vscode.window.showQuickPick(
        [
          { label: 'Erneut eingeben', value: 'retry' },
          { label: 'Ohne Variante fortfahren', value: 'skip' },
        ],
        {
          placeHolder: 'Leere Variante wird nicht gespeichert. Was möchtest du tun?',
        }
      );

      if (!emptyAction) {
        return undefined;
      }

      if (emptyAction.value === 'retry') {
        continue;
      }

      if (variants.length === 0) {
        return [];
      }

      break;
    }

    const variant: ExerciseVariantResult = {};
    if (variantName.trim() !== '') {
      variant.name = variantName.trim();
    }
    if (variantDescription.trim() !== '') {
      variant.description = variantDescription.trim();
    }
    if (Object.keys(trackingProperties).length > 0) {
      variant.tracking_properties = trackingProperties;
    }
    variants.push(variant);

    const addMore = await vscode.window.showQuickPick(
      [
        { label: 'Ja', value: 'yes' },
        { label: 'Nein', value: 'no' },
      ],
      { placeHolder: 'Noch eine Variante hinzufügen?' }
    );

    if (!addMore) {
      return undefined;
    }
    if (addMore.value === 'no') {
      break;
    }
  }

  return variants;
}

async function promptTrackingPropertiesForVariant(): Promise<Record<string, string> | undefined> {
  const properties: Record<string, string> = {};

  while (true) {
    const propName = await vscode.window.showInputBox({
      prompt: 'Tracking-Property (optional)',
      placeHolder: 'z.B. distance_km, reps, tempo_bpm (leer lassen zum Beenden)',
    });

    if (propName === undefined) {
      return undefined;
    }

    const cleanName = propName.trim();
    if (cleanName === '') {
      break;
    }

    const unit = await vscode.window.showInputBox({
      prompt: `Einheit für ${cleanName}`,
      value: 'text',
      placeHolder: 'z.B. km, min, kg, bpm, text',
    });

    if (unit === undefined) {
      return undefined;
    }

    properties[cleanName] = unit.trim() || 'text';
  }

  return properties;
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
