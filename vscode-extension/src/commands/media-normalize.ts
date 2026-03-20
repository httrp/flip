import * as vscode from 'vscode';
import { getFlipClient, BrainInfo, HealthIssue, HealthReport } from '../flip-client';

let output: vscode.OutputChannel | undefined;
function getOutput(): vscode.OutputChannel {
  if (!output) { output = vscode.window.createOutputChannel('Flip Media'); }
  return output;
}
export function disposeMediaOutput(): void { output?.dispose(); output = undefined; }

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
    summaryParts.push(`${repairs.stats.Repaired} repariert`);
    if (repairs.stats.Failed > 0) summaryParts.push(`${repairs.stats.Failed} fehlgeschlagen`);
    if (repairs.not_repairable_count > 0) summaryParts.push(`${repairs.not_repairable_count} nicht reparierbar`);
  }

  const title = `Media Normalize: ${check.BrainInfo.Name}`;
  getOutput().clear();
  getOutput().appendLine(title);
  getOutput().appendLine(`Typ: ${check.BrainInfo.Type}`);
  getOutput().appendLine(`Path: ${check.BrainInfo.Path}`);
  getOutput().appendLine(`Media: Assets=${check.Stats.AssetsChecked}`);
  getOutput().appendLine('');

  if (issues.length === 0) {
    getOutput().appendLine('✅ Keine Media-Issues gefunden');
  } else {
    getOutput().appendLine('Media Issues:');
    for (const issue of issues) {
      getOutput().appendLine(formatIssue(issue));
    }
  }

  if (repairs) {
    getOutput().appendLine('');
    getOutput().appendLine('Repairs:');
    for (const res of repairs.results) {
      const icon = res.Success ? '✓' : '✗';
      const line = res.Issue.Line > 0 ? `:${res.Issue.Line}` : '';
      getOutput().appendLine(`${icon} ${res.Issue.File}${line} → ${res.Message || res.SkipReason}`);
    }
    getOutput().appendLine('');
    getOutput().appendLine(`Stats: ${repairs.stats.Repaired} repaired, ${repairs.stats.Failed} failed, ${repairs.stats.Skipped} skipped`);
  }

  getOutput().show();

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
