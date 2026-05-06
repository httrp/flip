import * as vscode from 'vscode';
import * as path from 'path';
import {
  ExportConvertResult,
  ExportTemplate,
  ExportTemplateListResult,
  getFlipClient,
} from '../flip-client';

type ExportFormat = 'pdf' | 'html' | 'docx';

interface FormatPick extends vscode.QuickPickItem {
  value: ExportFormat;
}

interface TemplatePick extends vscode.QuickPickItem {
  value: string;
}

/**
 * Export current markdown file via flip export convert.
 */
export async function exportMarkdown(): Promise<void> {
  const client = getFlipClient();

  const inputFile = await selectInputFile();
  if (!inputFile) {
    return;
  }

  const format = await selectFormat();
  if (!format) {
    return;
  }

  const doctor = await client.exportDoctor(format);
  if (!doctor.success || !doctor.data?.pandoc_available) {
    const hints = doctor.data?.install_hints || [];
    const message = hints.length > 0
      ? `Pandoc missing. ${hints[0]}`
      : 'Pandoc missing. Run "flip export doctor" in terminal for install hints.';

    const action = await vscode.window.showErrorMessage(
      message,
      'Run Export Doctor'
    );

    if (action === 'Run Export Doctor') {
      await exportDoctor();
    }
    return;
  }

  const templateSelection = await selectTemplate(format);
  if (templateSelection === undefined) {
    return;
  }

  let landscape = false;
  if (format === 'pdf') {
    const layout = await vscode.window.showQuickPick(
      [
        { label: 'Portrait', value: false, description: 'Default' },
        { label: 'Landscape', value: true, description: 'Wider tables and slides' },
      ],
      { placeHolder: 'Choose PDF page orientation' }
    );
    if (!layout) {
      return;
    }
    landscape = layout.value;
  }

  const config = vscode.workspace.getConfiguration('flip');
  const openAfterDefault = config.get<boolean>('export.openAfterExport', true);
  const openSelection = await vscode.window.showQuickPick(
    [
      { label: 'Open after export', value: true, description: openAfterDefault ? 'Default' : undefined },
      { label: 'Do not open after export', value: false, description: !openAfterDefault ? 'Default' : undefined },
    ],
    { placeHolder: 'Open exported file automatically?' }
  );

  if (!openSelection) {
    return;
  }

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Exporting markdown...',
      cancellable: false,
    },
    async () => {
      return client.exportConvert({
        input: inputFile,
        format,
        template: format !== 'docx' ? templateSelection : undefined,
        referenceDoc: format === 'docx' ? templateSelection : undefined,
        landscape,
        openAfter: openSelection.value,
      });
    }
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Export failed: ${result.error || 'Unknown error'}`);
    return;
  }

  const data = result.data as ExportConvertResult;
  const openNow = await vscode.window.showInformationMessage(
    `Exported ${format.toUpperCase()} to ${data.output}`,
    'Open File',
    'Open Folder'
  );

  if (openNow === 'Open File') {
    await openPath(data.output);
  }
  if (openNow === 'Open Folder') {
    await openPath(path.dirname(data.output));
  }
}

/**
 * Run dependency checks for export and show a concise report.
 */
export async function exportDoctor(): Promise<void> {
  const client = getFlipClient();
  const format = await selectFormat('Which format should be checked?');
  if (!format) {
    return;
  }

  const result = await client.exportDoctor(format);
  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Export doctor failed: ${result.error || 'Unknown error'}`);
    return;
  }

  const data = result.data;
  const lines: string[] = [];
  lines.push(`# Flip Export Doctor (${format.toUpperCase()})`);
  lines.push('');
  lines.push(`- Pandoc: ${data.pandoc_available ? `OK (${data.pandoc_path || 'found'})` : 'missing'}`);
  if (format === 'pdf') {
    lines.push(`- XeLaTeX: ${data.xelatex_available ? `OK (${data.xelatex_path || 'found'})` : 'missing'}`);
  }
  lines.push(`- Default output dir: ${data.output_dir_default}`);
  lines.push(`- Templates found: ${data.template_count}`);

  if (data.install_hints && data.install_hints.length > 0) {
    lines.push('');
    lines.push('## Install Hints');
    for (const hint of data.install_hints) {
      lines.push(`- ${hint}`);
    }
  }

  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: lines.join('\n'),
  });
  await vscode.window.showTextDocument(doc, { preview: true });
}

async function selectInputFile(): Promise<string | undefined> {
  const editor = vscode.window.activeTextEditor;
  if (editor && editor.document.languageId === 'markdown') {
    const current = await vscode.window.showQuickPick(
      [
        { label: 'Use active markdown file', value: 'active', description: vscode.workspace.asRelativePath(editor.document.uri) },
        { label: 'Pick a markdown file...', value: 'pick' },
      ],
      { placeHolder: 'Select file to export' }
    );

    if (!current) {
      return undefined;
    }
    if (current.value === 'active') {
      return editor.document.uri.fsPath;
    }
  }

  const picked = await vscode.window.showOpenDialog({
    canSelectMany: false,
    filters: { Markdown: ['md'] },
    openLabel: 'Export this markdown file',
  });

  if (!picked || picked.length === 0) {
    return undefined;
  }
  return picked[0].fsPath;
}

async function selectFormat(placeHolder = 'Select export format'): Promise<ExportFormat | undefined> {
  const config = vscode.workspace.getConfiguration('flip');
  const defaultFormat = (config.get<string>('export.defaultFormat', 'pdf') || 'pdf').toLowerCase();

  const items: FormatPick[] = [
    { label: 'PDF', description: defaultFormat === 'pdf' ? 'Default' : undefined, value: 'pdf' },
    { label: 'HTML', description: defaultFormat === 'html' ? 'Default' : undefined, value: 'html' },
    { label: 'DOCX', description: defaultFormat === 'docx' ? 'Default' : undefined, value: 'docx' },
  ];

  const picked = await vscode.window.showQuickPick(items, { placeHolder });
  return picked?.value;
}

async function selectTemplate(format: ExportFormat): Promise<string | undefined> {
  const client = getFlipClient();
  const result = await client.listExportTemplates(format);

  const templates: ExportTemplate[] = (result.success && result.data)
    ? (result.data as ExportTemplateListResult).templates || []
    : [];

  const noneLabel = format === 'docx' ? 'No reference DOCX' : 'Auto template';
  const picks: TemplatePick[] = [{ label: noneLabel, description: 'Recommended default', value: '' }];

  for (const t of templates) {
    picks.push({
      label: t.name,
      description: `${t.kind} | ${t.source}`,
      detail: t.path,
      value: t.path,
    });
  }

  const selected = await vscode.window.showQuickPick(picks, {
    placeHolder: format === 'docx'
      ? 'Choose reference DOCX (optional)'
      : 'Choose template (optional)',
  });

  if (!selected) {
    return undefined;
  }
  return selected.value;
}

async function openPath(path: string): Promise<void> {
  const uri = vscode.Uri.file(path);
  const ext = path.toLowerCase().split('.').pop();

  if (ext === 'html' || ext === 'md' || ext === 'txt') {
    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc, { preview: true });
      return;
    } catch {
      // Fallback to system handler below.
    }
  }

  await vscode.env.openExternal(uri);
}
