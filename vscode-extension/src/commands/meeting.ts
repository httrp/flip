import * as vscode from 'vscode';
import { getFlipClient, NoteResult, MeetingSeriesInfo } from '../flip-client';

/**
 * Create a meeting note with streamlined flow:
 * 1. Select brain
 * 2. Enter organization (optional)
 * 3. Part of series? → select existing or create new
 * 4. Auto-generate title for series, or prompt for single meeting
 */
export async function createMeetingNote(): Promise<void> {
  const client = getFlipClient();

  // Step 1: Choose brain
  const info = await client.getInfo();
  let brain: string | undefined;
  
  if (!info.success || !info.data || info.data.brains.length === 0) {
    vscode.window.showErrorMessage('No brains found. Please add a brain first.');
    return;
  }

  if (info.data.brains.length > 1) {
    const pick = await vscode.window.showQuickPick(
      info.data.brains.map((b) => ({ 
        label: b.name, 
        description: b.active ? '(active)' : '',
        detail: b.path 
      })),
      { placeHolder: 'Select brain for meeting note' }
    );
    if (!pick) return;
    brain = pick.label;
  } else {
    brain = info.data.brains[0].name;
  }

  // Step 2: Organization (optional)
  const organization = await vscode.window.showInputBox({
    prompt: 'Organization (optional, press Enter to skip)',
    placeHolder: 'e.g., P1174, DANORAMA',
  });
  if (organization === undefined) return; // cancelled

  // Step 3: Part of a series?
  const seriesChoice = await vscode.window.showQuickPick(
    [
      { label: '$(file) Single meeting', value: 'single' },
      { label: '$(list-tree) Part of a series', value: 'series' },
    ],
    { placeHolder: 'Is this part of a meeting series?' }
  );
  if (!seriesChoice) return;

  let title: string;
  let series: string | undefined;

  if (seriesChoice.value === 'series') {
    // Load existing series
    const seriesResult = await client.listMeetingSeries();
    let existingSeries: MeetingSeriesInfo[] = [];
    
    if (seriesResult.success && seriesResult.data) {
      existingSeries = seriesResult.data.series || [];
    }

    // Build options: existing series + "Create new"
    const seriesOptions: { label: string; description?: string; value: string }[] = [
      { label: '$(add) Create new series', value: '__new__' },
    ];
    
    for (const s of existingSeries) {
      seriesOptions.push({
        label: s.name,
        description: `${s.count} meetings`,
        value: s.name,
      });
    }

    const selectedSeries = await vscode.window.showQuickPick(seriesOptions, {
      placeHolder: 'Select series or create new',
    });
    if (!selectedSeries) return;

    if (selectedSeries.value === '__new__') {
      // Create new series
      const newSeriesName = await vscode.window.showInputBox({
        prompt: 'Series name',
        placeHolder: 'e.g., Weekly Standup, Sprint Planning',
        validateInput: (v) => (!v || v.trim() === '' ? 'Series name is required' : null),
      });
      if (!newSeriesName) return;
      series = newSeriesName.trim();
    } else {
      series = selectedSeries.value;
    }

    // Auto-generate title: "Series Name - DD.MM.YYYY"
    const now = new Date();
    const day = String(now.getDate()).padStart(2, '0');
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const year = now.getFullYear();
    title = `${series} - ${day}.${month}.${year}`;
  } else {
    // Single meeting: prompt for title
    const inputTitle = await vscode.window.showInputBox({
      prompt: 'Meeting title',
      placeHolder: 'Sprint Planning',
      validateInput: (v) => (!v || v.trim() === '' ? 'Title is required' : null),
    });
    if (!inputTitle) return;
    title = inputTitle.trim();
  }

  // Create the meeting note
  await vscode.window.withProgress(
    { location: vscode.ProgressLocation.Notification, title: 'Creating meeting note...', cancellable: false },
    async () => {
      const res = await client.createMeeting({ 
        title, 
        organization: organization || undefined,
        series,
        brain 
      });
      
      if (!res.success) {
        vscode.window.showErrorMessage(`Failed to create meeting note: ${res.error}`);
        return;
      }
      
      const data = res.data as NoteResult;
      const doc = await vscode.workspace.openTextDocument(data.path);
      await vscode.window.showTextDocument(doc);

      const choice = await vscode.window.showInformationMessage(
        `Created meeting note in ${data.brain_name}. Link in today's journal?`,
        'Yes',
        'No'
      );
      if (choice === 'Yes') {
        const linkRes = await client.addToJournal({ 
          file: data.path, 
          title: data.title, 
          type: 'meeting', 
          brain: data.brain_name 
        });
        if (!linkRes.success) {
          vscode.window.showWarningMessage(`Failed to add to journal: ${linkRes.error}`);
        } else {
          vscode.window.showInformationMessage("✓ Linked in today's journal");
        }
      }
    }
  );
}
