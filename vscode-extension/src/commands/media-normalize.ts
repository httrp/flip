import * as vscode from 'vscode';
import { getFlipClient, BrainInfo, HealthIssue, HealthReport } from '../flip-client';

const output = vscode.window.createOutputChannel('Flip Media');

export async function mediaNormalize(): Promise<void> {
  const client = getFlipClient();

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

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = brains.map((b) => ({
    label: `$(database) ${b.name}`,
    description: b.active ? '(aktiv)' : '',
    detail: b.path,
    brain: b,
  }));

  brainItems.sort((a, b) => {
    if (a.brain.active) return -1;
    if (b.brain.active) return 1;
    return 0;
  });

  const selectedBrain = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Brain für Media Normalize auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  const mode = await vscode.window.showQuickPick(
    [
      { label: 'Nur prüfen', description: 'Keine Änderungen', value: 'check' },
      { label: 'Normalisieren (mit Vorschau)', description: 'Dry-run, zeigt geplante Änderungen', value: 'preview' },
      { label: 'Normalisieren (anwenden)', description: 'Umbenennen/Verschieben und Links aktualisieren', value: 'fix' },
    ],
    { placeHolder: 'Media Normalize Modus wählen' }
  );

  if (!mode) {
    return;
  }

  const fix = mode.value === 'fix' || mode.value === 'preview';
  const dryRun = mode.value === 'preview';

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Normalisiere Medien in "${selectedBrain.brain.name}"...`,
      cancellable: false,
    },
    async () => {
      return await client.normalizeMedia({ brainPath: selectedBrain.brain.path, fix, dryRun });
    }
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Media Normalize fehlgeschlagen: ${result.error}`);
    return;
  }

  const report: HealthReport = result.data;
  const check = report.result;
  if (!check) {
    vscode.window.showErrorMessage('Media Normalize lieferte kein Ergebnis');
    return;
  }

  const issues = check.Issues || [];
  const errors = issues.filter((i) => i.Severity === 'error').length;
  const warnings = issues.filter((i) => i.Severity === 'warning').length;
  const infos = issues.filter((i) => i.Severity === 'info').length;

  const repairs = report.repairs;

  const summaryParts: string[] = [];
  summaryParts.push(`${errors} Fehler`, `${warnings} Warnungen`, `${infos} Hinweise`);
  if (repairs) {
    summaryParts.push(`${repairs.Stats.Repaired} repariert`);
    if (repairs.Stats.Failed > 0) summaryParts.push(`${repairs.Stats.Failed} fehlgeschlagen`);
    if (repairs.NotRepairableCount > 0) summaryParts.push(`${repairs.NotRepairableCount} nicht reparierbar`);
  }

  const title = `Media Normalize: ${check.BrainInfo.Name}`;
  output.clear();
  output.appendLine(title);
  output.appendLine(`Typ: ${check.BrainInfo.Type}`);
  output.appendLine(`Path: ${check.BrainInfo.Path}`);
  output.appendLine(`Media: Assets=${check.Stats.AssetsChecked}`);
  output.appendLine('');

  if (issues.length === 0) {
    output.appendLine('✅ Keine Media-Issues gefunden');
  } else {
    output.appendLine('Media Issues:');
    for (const issue of issues) {
      output.appendLine(formatIssue(issue));
    }
  }

  if (repairs) {
    output.appendLine('');
    output.appendLine('Repairs:');
    for (const res of repairs.Results) {
      const icon = res.Success ? '✓' : '✗';
      const line = res.Issue.Line > 0 ? `:${res.Issue.Line}` : '';
      output.appendLine(`${icon} ${res.Issue.File}${line} → ${res.Message || res.SkipReason}`);
    }
    output.appendLine('');
    output.appendLine(`Stats: ${repairs.Stats.Repaired} repaired, ${repairs.Stats.Failed} failed, ${repairs.Stats.Skipped} skipped`);
  }

  output.show();

  const summary = summaryParts.join(', ');
  if (dryRun) {
    vscode.window.showInformationMessage(`[DRY-RUN] ${summary}`);
  } else if (fix) {
    vscode.window.showInformationMessage(`Media normalisiert: ${summary}`);
  } else {
    vscode.window.showInformationMessage(`Media Check: ${summary}`);
  }
}

function formatIssue(issue: HealthIssue): string {
  const line = issue.Line > 0 ? `:${issue.Line}` : '';
  let severity = issue.Severity.toUpperCase();
  if (issue.Severity === 'error') severity = 'ERROR';
  if (issue.Severity === 'warning') severity = 'WARN';
  if (issue.Severity === 'info') severity = 'INFO';
  return `[${severity}] ${issue.File}${line} → ${issue.Message} – ${issue.Details}`;
}
