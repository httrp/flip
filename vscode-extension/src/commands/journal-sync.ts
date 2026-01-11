import * as vscode from 'vscode';
import { getFlipClient, MissingJournalEntry, DayResult, BrainInfo } from '../flip-client';

/**
 * Check and sync missing journal entries
 * Shows files that were created/modified recently but aren't in the journal
 */
export async function journalSyncCheck(): Promise<void> {
  const client = getFlipClient();

  // First, get available brains
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

  // Let user select a brain
  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains.map((b: BrainInfo) => ({
    label: `$(database) ${b.name}`,
    description: b.active ? '(aktiv)' : '',
    detail: b.path,
    brain: b,
  }));

  // Put active brain first
  brainItems.sort((a: BrainPickItem, b: BrainPickItem) => {
    if (a.brain.active) return -1;
    if (b.brain.active) return 1;
    return 0;
  });

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Brain für Journal-Prüfung auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  // Ask user for number of days
  const daysChoice = await vscode.window.showQuickPick(
    [
      { label: '$(calendar) Letzte 3 Tage', days: 3 },
      { label: '$(calendar) Letzte 7 Tage', days: 7 },
      { label: '$(calendar) Letzte 14 Tage', days: 14 },
      { label: '$(calendar) Letzte 30 Tage', days: 30 },
    ],
    { placeHolder: 'Zeitraum für Journal-Prüfung wählen' }
  );

  if (!daysChoice) {
    return;
  }

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Prüfe Journal in "${selectedBrain.brain.name}" (${daysChoice.days} Tage)...`,
      cancellable: false,
    },
    async () => {
      return await client.journalSyncCheck(daysChoice.days, selectedBrain.brain.name);
    }
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Journal-Prüfung fehlgeschlagen: ${result.error}`);
    return;
  }

  const data = result.data;

  if (data.total_missing === 0) {
    vscode.window.showInformationMessage(
      `✓ Journal ist vollständig! (${data.total_linked} verlinkt, ${data.total_ignored} ignoriert, ${daysChoice.days} Tage geprüft)`
    );
    return;
  }

  // Build quick pick items grouped by day
  interface PickItem extends vscode.QuickPickItem {
    entry: MissingJournalEntry | null;
    day: DayResult | null;
    action: 'single' | 'day' | 'all';
  }

  const items: PickItem[] = [];

  // Add "All" option at the top
  items.push({
    label: '$(checklist) Alle hinzufügen',
    description: `${data.total_missing} Einträge`,
    detail: 'Fügt alle fehlenden Einträge zum jeweiligen Journal hinzu',
    entry: null,
    day: null,
    action: 'all',
  });

  // Group by day
  for (const day of data.days) {
    if (day.missing_entries.length === 0) {
      continue;
    }

    // Day separator
    const dayLabel = formatDate(day.date);
    items.push({
      label: `$(calendar) ${dayLabel}`,
      description: `${day.missing_entries.length} fehlend`,
      detail: day.journal_exists ? `Journal: ${day.journal_path}` : '⚠️ Journal existiert noch nicht',
      kind: vscode.QuickPickItemKind.Separator,
      entry: null,
      day: day,
      action: 'day',
    });

    // Entries for this day
    for (const entry of day.missing_entries) {
      items.push({
        label: `    $(${getIconForType(entry.type)}) ${entry.title}`,
        description: entry.modified,
        detail: entry.rel_path,
        entry,
        day,
        action: 'single',
      });
    }
  }

  const selected = await vscode.window.showQuickPick(items.filter(i => i.kind !== vscode.QuickPickItemKind.Separator), {
    placeHolder: `${data.total_missing} Einträge fehlen im Journal - auswählen zum Hinzufügen`,
    canPickMany: true,
  });

  if (!selected || selected.length === 0) {
    return;
  }

  // Determine which entries to add
  let entriesToAdd: { entry: MissingJournalEntry; journalPath: string }[] = [];
  
  if (selected.some((s) => s.action === 'all')) {
    // "All" was selected - add all entries
    for (const day of data.days) {
      for (const entry of day.missing_entries) {
        entriesToAdd.push({ entry, journalPath: day.journal_path });
      }
    }
  } else {
    // Add selected entries
    for (const s of selected) {
      if (s.entry && s.day) {
        entriesToAdd.push({ entry: s.entry, journalPath: s.day.journal_path });
      }
    }
  }

  // Add entries to journal with progress
  let successCount = 0;
  let failedEntries: string[] = [];

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Füge Einträge zum Journal hinzu...',
      cancellable: false,
    },
    async (progress) => {
      const total = entriesToAdd.length;
      for (let i = 0; i < entriesToAdd.length; i++) {
        const { entry } = entriesToAdd[i];
        progress.report({ 
          message: `(${i + 1}/${total}) ${entry.title}`,
          increment: 100 / total 
        });

        const addResult = await client.addToJournal({
          file: entry.path,
          title: entry.title,
          type: entry.type as 'note' | 'meeting' | 'task',
          brain: data.brain_name,
          date: entry.date,
        });

        if (addResult.success) {
          successCount++;
        } else {
          failedEntries.push(entry.title);
        }
      }
    }
  );

  // Show result message
  if (failedEntries.length === 0) {
    vscode.window.showInformationMessage(
      `✅ ${successCount} Einträge erfolgreich zum Journal hinzugefügt`
    );
  } else {
    vscode.window.showWarningMessage(
      `⚠️ ${successCount}/${entriesToAdd.length} hinzugefügt. Fehlgeschlagen: ${failedEntries.join(', ')}`
    );
  }
}

/**
 * Format date for display
 */
function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (dateStr === today.toISOString().split('T')[0]) {
    return 'Heute';
  }
  if (dateStr === yesterday.toISOString().split('T')[0]) {
    return 'Gestern';
  }
  
  // Format as "Mo, 09.01."
  const days = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa'];
  const day = days[date.getDay()];
  return `${day}, ${date.getDate().toString().padStart(2, '0')}.${(date.getMonth() + 1).toString().padStart(2, '0')}.`;
}

/**
 * Get VS Code icon for entry type
 */
function getIconForType(type: string): string {
  switch (type) {
    case 'meeting':
      return 'organization';
    case 'task':
      return 'tasklist';
    default:
      return 'note';
  }
}
