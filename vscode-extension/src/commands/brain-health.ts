import * as vscode from 'vscode';
import { getFlipClient, BrainInfo, HealthIssue, HealthReport } from '../flip-client';

const output = vscode.window.createOutputChannel('Flip Health');

export async function brainHealthCheck(): Promise<void> {
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
    placeHolder: 'Brain für Health Check auswählen',
  });

  if (!selectedBrain) {
    return;
  }

  const mode = await vscode.window.showQuickPick(
    [
      { label: 'Nur prüfen', description: 'Keine Änderungen', value: 'check' },
      { label: 'Reparieren (mit Vorschau)', description: 'Dry-run, zeigt geplante Änderungen', value: 'preview' },
      { label: 'Reparieren (anwenden)', description: 'Versucht reparierbare Issues zu beheben', value: 'fix' },
    ],
    { placeHolder: 'Health Check Modus wählen' }
  );

  if (!mode) {
    return;
  }

  const fix = mode.value === 'fix' || mode.value === 'preview';
  const dryRun = mode.value === 'preview';

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Prüfe Brain "${selectedBrain.brain.name}"...`,
      cancellable: false,
    },
    async () => {
      return await client.checkBrainHealth({ brainPath: selectedBrain.brain.path, fix, dryRun });
    }
  );

  // Debug logging
  output.clear();
  output.appendLine('=== DEBUG INFO ===');
  output.appendLine(`Result success: ${result.success}`);
  output.appendLine(`Result command: ${result.command}`);
  output.appendLine(`Result data exists: ${!!result.data}`);
  output.appendLine(`Result error: ${result.error || 'none'}`);
  output.appendLine(`Raw result: ${JSON.stringify(result, null, 2)}`);
  output.appendLine('==================');
  output.show(true);

  if (!result.success || !result.data) {
    const errorMsg = result.error ? (typeof result.error === 'string' ? result.error : JSON.stringify(result.error)) : 'Unbekannter Fehler';
    vscode.window.showErrorMessage(`Health Check fehlgeschlagen: ${errorMsg}`);
    return;
  }

  const report: HealthReport = result.data;
  const check = report.result;
  if (!check) {
    vscode.window.showErrorMessage('Health Check lieferte kein Ergebnis');
    return;
  }

  const issues = check.Issues || [];
  const errors = issues.filter((i) => i.Severity === 'error').length;
  const warnings = issues.filter((i) => i.Severity === 'warning').length;

  const repairs = report.repairs;

  const summaryParts: string[] = [];
  summaryParts.push(`${errors} Fehler`, `${warnings} Warnungen`);
  if (repairs) {
    summaryParts.push(`${repairs.Stats.Repaired} repariert`);
    if (repairs.Stats.Failed > 0) summaryParts.push(`${repairs.Stats.Failed} fehlgeschlagen`);
    if (repairs.NotRepairableCount > 0) summaryParts.push(`${repairs.NotRepairableCount} nicht reparierbar`);
  }

  const title = `Brain Health: ${check.BrainInfo.Name}`;
  output.clear();
  output.appendLine(title);
  output.appendLine(`Typ: ${check.BrainInfo.Type}`);
  output.appendLine(`Path: ${check.BrainInfo.Path}`);
  output.appendLine(`Scan: Files=${check.Stats.FilesScanned}, Links=${check.Stats.LinksChecked}, Assets=${check.Stats.AssetsChecked}`);
  output.appendLine('');

  if (issues.length === 0) {
    output.appendLine('✅ Keine Issues gefunden');
  } else {
    output.appendLine('Issues:');
    for (const issue of issues) {
      output.appendLine(formatIssue(issue));
    }
  }

  if (repairs) {
    output.appendLine('');
    output.appendLine('Repairs:');
    for (const res of repairs.Results) {
      const icon = res.Success ? '✓' : '✗';
      output.appendLine(` ${icon} ${res.Issue.File}`);
      if (res.Message && res.Message !== 'Repaired successfully') {
        output.appendLine(`    ${res.Message}`);
      }
      if (res.SkipReason) {
        output.appendLine(`    Skipped: ${res.SkipReason}`);
      }
    }
  }

  output.show(true);

  const message = repairs
    ? `${title} – ${summaryParts.join(', ')}${dryRun ? ' (Dry-Run)' : ''}`
    : `${title} – ${errors} Fehler, ${warnings} Warnungen`;

  if (errors === 0 && (!repairs || repairs.Stats.Failed === 0)) {
    vscode.window.showInformationMessage(message, 'Output öffnen').then((choice) => {
      if (choice === 'Output öffnen') output.show(true);
    });
  } else {
    vscode.window.showWarningMessage(message, 'Output öffnen').then((choice) => {
      if (choice === 'Output öffnen') output.show(true);
    });
  }
}

function formatIssue(issue: HealthIssue): string {
  const location = issue.Line ? `${issue.File}:${issue.Line}` : issue.File;
  const details = issue.Details ? ` – ${issue.Details}` : '';
  return ` [${issue.Severity}] ${location} → ${issue.Message}${details}`;
}
