import * as vscode from 'vscode';
import { getFlipClient, NoteResult, MeetingSeriesInfo, DefinitionItem } from '../flip-client';
import { promptAutoAddToJournal } from './journal-helper';

/**
 * Create a meeting note with streamlined flow:
 * 1. Select brain
 * 2. Select organization from definitions (optional)
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

  // Step 2: Organization from definitions (optional)
  let organization: string | undefined;
  const defsResult = await client.listDefinitions();
  const orgItems: { label: string; description?: string; value: string }[] = [
    { label: '$(circle-slash) Ohne Organisation', description: 'überspringen', value: '__none__' },
    { label: '$(add) Neue Organisation', description: 'manuell eingeben', value: '__new__' },
  ];

  if (defsResult.success && defsResult.data?.organizations?.length) {
    (defsResult.data.organizations as DefinitionItem[]).forEach((org) => {
      const desc = org.description || org.name;
      orgItems.push({ 
        label: `[${org.abbreviation}] ${org.name}`, 
        description: desc !== org.name ? desc : undefined, 
        value: org.abbreviation 
      });
    });
  }

  const orgPick = await vscode.window.showQuickPick(orgItems, {
    placeHolder: 'Organisation wählen oder neu anlegen',
    ignoreFocusOut: true,
  });
  if (!orgPick) return; // cancelled

  if (orgPick.value === '__new__') {
    const newOrgAbbr = await vscode.window.showInputBox({
      prompt: 'Organisation Kürzel',
      placeHolder: 'z.B. P1174, DANO',
      validateInput: (v) => (!v || v.trim() === '' ? 'Kürzel darf nicht leer sein' : null),
    });
    if (newOrgAbbr === undefined) return; // cancelled
    
    const newOrgName = await vscode.window.showInputBox({
      prompt: 'Organisation Name (optional)',
      placeHolder: 'z.B. Projekt 1174, DANORAMA GmbH',
    });
    if (newOrgName === undefined) return; // cancelled
    
    organization = newOrgAbbr.trim().toUpperCase();
    
    // Save new org to definitions
    const addResult = await client.addOrganization({
      abbreviation: organization,
      name: newOrgName?.trim() || undefined,
    });
    if (addResult.success) {
      vscode.window.showInformationMessage(`Organisation [${organization}] angelegt`);
    } else {
      // Don't block if org already exists, just continue
      if (!addResult.error?.includes('already exists')) {
        vscode.window.showWarningMessage(`Organisation konnte nicht gespeichert werden: ${addResult.error}`);
      }
    }
  } else if (orgPick.value !== '__none__') {
    organization = orgPick.value;
  }

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
  const res = await vscode.window.withProgress(
    { location: vscode.ProgressLocation.Notification, title: 'Creating meeting note...', cancellable: false },
    async () => {
      return await client.createMeeting({ 
        title, 
        organization: organization || undefined,
        series,
        brain 
      });
    }
  );
  
  if (!res.success) {
    vscode.window.showErrorMessage(`Failed to create meeting note: ${res.error}`);
    return;
  }
  
  const data = res.data as NoteResult;
  const doc = await vscode.workspace.openTextDocument(data.path);
  await vscode.window.showTextDocument(doc);

  // Auto-add to journal with countdown (default = add)
  await promptAutoAddToJournal(data, 'meeting');
}
