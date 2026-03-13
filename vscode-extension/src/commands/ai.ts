import * as vscode from 'vscode';
import * as path from 'path';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { getFlipClient, BrainInfo, getFlipExecutablePath, NoteResult, PromptsResult } from '../flip-client';

const execFileAsync = promisify(execFile);

interface AIModelInfo {
  id: string;
  name?: string;
  description?: string;
  context_size?: number;
}

interface AIModelsResponse {
  provider: string;
  default_model: string;
  models: AIModelInfo[];
}

interface PromptSelection {
  mode: 'none' | 'existing' | 'create';
  value?: string;
  title?: string;
  body?: string;
  setDefault?: boolean;
}

let aiTerminal: vscode.Terminal | undefined;

function getAiTerminal(): vscode.Terminal {
  if (!aiTerminal) {
    aiTerminal = vscode.window.createTerminal('Flip AI');
  }
  return aiTerminal;
}

function runFlipAiCommand(args: string[]): void {
  const terminal = getAiTerminal();
  const cmd = ['flip', 'ai', ...args].join(' ');
  terminal.show(true);
  terminal.sendText(cmd);
}

function getAiTimeoutMs(): number {
  const config = vscode.workspace.getConfiguration('flip');
  const timeoutSec = config.get<number>('aiTimeout', 180);
  return Math.max(30, timeoutSec) * 1000;
}

function findBrainForFile(filePath: string, brains: BrainInfo[]): BrainInfo | undefined {
  const normalized = path.normalize(filePath);
  let match: BrainInfo | undefined;
  let longest = -1;

  for (const brain of brains) {
    const brainPath = path.normalize(brain.path);
    if (normalized === brainPath || normalized.startsWith(brainPath + path.sep)) {
      if (brainPath.length > longest) {
        longest = brainPath.length;
        match = brain;
      }
    }
  }

  return match;
}

async function promptMultiLineInput(title: string, initialContent: string): Promise<string | undefined> {
  const doc = await vscode.workspace.openTextDocument({
    language: 'markdown',
    content: initialContent,
  });

  await vscode.window.showTextDocument(doc, { preview: false });

  const choice = await vscode.window.showInformationMessage(
    title,
    { modal: true, detail: 'Edit the text, then choose “Use content”.' },
    'Use content',
    'Cancel'
  );

  if (choice !== 'Use content') {
    return undefined;
  }

  return doc.getText();
}

async function promptExtraInstructions(): Promise<string | undefined> {
  const choice = await vscode.window.showQuickPick(
    [
      { label: 'No extra instructions', value: 'none' },
      { label: 'Add extra instructions (editor)', value: 'editor' },
    ],
    { placeHolder: 'Add extra instructions for this request?' }
  );

  if (!choice) {
    return undefined;
  }

  if (choice.value === 'none') {
    return '';
  }

  const content = await promptMultiLineInput('Extra instructions', '');
  if (content === undefined) {
    return undefined;
  }

  return content.trim();
}

function appendPromptArgs(args: string[], selection: PromptSelection, extra: string | undefined): void {
  if (selection.mode === 'existing' && selection.value) {
    args.push('--prompt', selection.value);
  } else if (selection.mode === 'create' && selection.title) {
    args.push('--prompt-create-title', selection.title);
    args.push('--prompt-create-body', selection.body || '');
    if (selection.setDefault) {
      args.push('--prompt-create-default');
    }
  } else {
    args.push('--prompt', 'none');
  }

  if (extra && extra.trim() !== '') {
    args.push('--prompt-extra', extra.trim());
  }
}

function suggestTitleFromTopic(topic: string): string {
  const cleaned = topic.trim();
  if (!cleaned) {
    return '';
  }

  const stopChars = ['.', '?', '!'];
  let candidate = cleaned;
  for (const char of stopChars) {
    const idx = candidate.indexOf(char);
    if (idx > 15) {
      candidate = candidate.slice(0, idx);
      break;
    }
  }

  const words = candidate.split(/\s+/).filter(Boolean);
  if (words.length > 8) {
    candidate = words.slice(0, 8).join(' ');
  }

  return candidate.replace(/^"|"$/g, '').trim();
}

async function selectPromptNote(brain: string): Promise<PromptSelection | undefined> {
  const client = getFlipClient();
  const result = await client.runCommand(['vscode', 'prompts', '--brain', brain]);

  if (!result.success || !result.data) {
    return { mode: 'none' };
  }

  const data = result.data as PromptsResult;
  const prompts = data.prompts || [];

  if (data.default_prompt) {
    return { mode: 'existing', value: data.default_prompt };
  }

  if (prompts.length === 0) {
    const choice = await vscode.window.showQuickPick(
      [
        { label: 'Create new prompt', value: 'create' },
        { label: 'No prompt', value: 'none' },
      ],
      { placeHolder: 'No prompt notes found' }
    );

    if (!choice) {
      return undefined;
    }

    if (choice.value === 'none') {
      return { mode: 'none' };
    }

    const title = await vscode.window.showInputBox({
      prompt: 'Prompt title',
      placeHolder: 'Research (Deep)',
    });
    if (title === undefined) {
      return undefined;
    }
    if (title.trim() === '') {
      vscode.window.showWarningMessage('Prompt title is required.');
      return { mode: 'none' };
    }

    const body = await promptMultiLineInput('Prompt instructions', '');
    if (body === undefined) {
      return undefined;
    }

    const setDefaultChoice = await vscode.window.showQuickPick(
      [
        { label: 'Set as default prompt', value: 'yes' },
        { label: 'Do not set default', value: 'no' },
      ],
      { placeHolder: 'Default prompt?' }
    );
    if (!setDefaultChoice) {
      return undefined;
    }

    return {
      mode: 'create',
      title: title.trim(),
      body: body.trim(),
      setDefault: setDefaultChoice.value === 'yes',
    };
  }

  const items: Array<vscode.QuickPickItem & { value: string }> = [
    { label: 'Create new prompt', value: 'create' },
    { label: 'No prompt', value: 'none' },
  ];

  for (const prompt of prompts) {
    items.push({
      label: prompt.title || prompt.name,
      description: prompt.rel_path,
      value: prompt.path,
    });
  }

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select prompt note (optional)',
  });

  if (!selected) {
    return undefined;
  }

  if (selected.value === 'create') {
    const title = await vscode.window.showInputBox({
      prompt: 'Prompt title',
      placeHolder: 'Research (Deep)',
    });
    if (title === undefined) {
      return undefined;
    }
    if (title.trim() === '') {
      vscode.window.showWarningMessage('Prompt title is required.');
      return { mode: 'none' };
    }

    const body = await promptMultiLineInput('Prompt instructions', '');
    if (body === undefined) {
      return undefined;
    }

    const setDefaultChoice = await vscode.window.showQuickPick(
      [
        { label: 'Set as default prompt', value: 'yes' },
        { label: 'Do not set default', value: 'no' },
      ],
      { placeHolder: 'Default prompt?' }
    );
    if (!setDefaultChoice) {
      return undefined;
    }

    return {
      mode: 'create',
      title: title.trim(),
      body: body.trim(),
      setDefault: setDefaultChoice.value === 'yes',
    };
  }

  if (selected.value === 'none') {
    return { mode: 'none' };
  }

  return { mode: 'existing', value: selected.value };
}

function scoreModel(action: 'research' | 'summarize' | 'improve', id: string): number {
  const name = id.toLowerCase();
  let score = 0;

  if (action === 'research') {
    if (name.includes('70b') || name.includes('opus') || name.includes('gpt-4') || name.includes('sonnet')) {
      score += 3;
    }
    if (name.includes('mini') || name.includes('haiku') || name.includes('8b') || name.includes('instant')) {
      score -= 1;
    }
  } else {
    if (name.includes('mini') || name.includes('haiku') || name.includes('8b') || name.includes('instant')) {
      score += 3;
    }
    if (name.includes('sonnet')) {
      score += 1;
    }
    if (name.includes('70b') || name.includes('opus') || name.includes('gpt-4')) {
      score -= 1;
    }
  }

  return score;
}

async function pickModel(action: 'research' | 'summarize' | 'improve'): Promise<string | null | undefined> {
  const client = getFlipClient();
  const result = await client.runCommand(['ai', 'models']);

  if (!result.success || !result.data) {
    vscode.window.showWarningMessage(`AI models not available: ${result.error || 'Unknown error'}`);
    return undefined;
  }

  const data = result.data as AIModelsResponse;
  const models = data.models || [];

  let recommendedId = '';
  let bestScore = -999;
  for (const model of models) {
    const id = model.id || model.name || '';
    if (!id) {
      continue;
    }
    const score = scoreModel(action, id);
    if (score > bestScore) {
      bestScore = score;
      recommendedId = id;
    }
  }

  const items: Array<vscode.QuickPickItem & { value: string }> = [
    {
      label: `Use default (${data.provider}: ${data.default_model})`,
      description: 'recommended default',
      value: '',
    },
  ];

  for (const model of models) {
    const id = model.id || model.name || '';
    if (!id) {
      continue;
    }
    const label = model.name && model.name !== id ? model.name : id;
    const details = [id !== label ? id : '', model.description || ''].filter(Boolean).join(' • ');
    const isRecommended = id === recommendedId;
    items.push({
      label: isRecommended ? `${label} (recommended)` : label,
      description: details || undefined,
      value: id,
    });
  }

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select model (optional)',
  });

  if (!selected) {
    return undefined;
  }

  return selected.value === '' ? null : selected.value;
}

async function selectBrain(): Promise<string | undefined> {
  const client = getFlipClient();
  const info = await client.getInfo();
  if (!info.success || !info.data || info.data.brains.length === 0) {
    vscode.window.showErrorMessage('No brains found. Please add a brain first.');
    return undefined;
  }

  if (info.data.brains.length === 1) {
    return info.data.brains[0].name;
  }

  interface BrainPickItem extends vscode.QuickPickItem {
    brain: BrainInfo;
  }

  const brainItems: BrainPickItem[] = info.data.brains.map((b) => ({
    label: b.name,
    description: b.active ? '(active)' : '',
    detail: `${b.type} - ${b.path}`,
    brain: b,
  }));

  const selected = await vscode.window.showQuickPick(brainItems, {
    placeHolder: 'Select brain for AI note',
  });

  return selected?.brain.name;
}

async function selectJournalLinkMode(): Promise<'link' | 'no-link' | undefined> {
  const selected = await vscode.window.showQuickPick(
    [
      { label: 'Add link to today\'s journal', value: 'link' },
      { label: 'Skip journal link', value: 'no-link' },
    ],
    { placeHolder: 'Journal link for AI note' }
  );

  return selected?.value as 'link' | 'no-link' | undefined;
}

export async function aiStatusCommand(): Promise<void> {
  runFlipAiCommand(['status']);
}

export async function aiResearchCommand(): Promise<void> {
  const client = getFlipClient();

  const brain = await selectBrain();
  if (!brain) {
    return;
  }

  const promptSelection = await selectPromptNote(brain);
  if (!promptSelection) {
    return;
  }

  const promptExtra = await promptExtraInstructions();
  if (promptExtra === undefined) {
    return;
  }

  const linkMode = await selectJournalLinkMode();
  if (!linkMode) {
    return;
  }

  const model = await pickModel('research');
  if (model === undefined) {
    return;
  }

  const topic = await vscode.window.showInputBox({
    prompt: 'Research topic (leave empty for interactive prompt)',
    placeHolder: 'Kubernetes networking',
  });

  if (topic === undefined) {
    return;
  }

  if (topic.trim() === '') {
    vscode.window.showWarningMessage('Topic is required for AI research.');
    return;
  }

  const suggestedTitle = suggestTitleFromTopic(topic.trim());
  const title = await vscode.window.showInputBox({
    prompt: 'Title',
    placeHolder: 'Short, clear title for the note',
    value: suggestedTitle,
  });

  if (title === undefined) {
    return;
  }

  if (title.trim() === '') {
    vscode.window.showWarningMessage('Title is required for AI research.');
    return;
  }

  const args: string[] = [
    'ai',
    'research',
    topic.trim(),
    '--no-stream',
    '--brain', brain,
    linkMode === 'link' ? '--link' : '--no-link',
  ];
  appendPromptArgs(args, promptSelection, promptExtra);
  if (model) {
    args.push('--model', model);
  }
  args.push('--title', title.trim());

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating research note...',
      cancellable: false,
    },
    async () => client.runCommand(args, getAiTimeoutMs())
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Failed to create research note: ${result.error}`);
    return;
  }

  const data = result.data as NoteResult;
  const doc = await vscode.workspace.openTextDocument(data.path);
  await vscode.window.showTextDocument(doc, { preview: false });
}

export async function aiSummarizeCommand(): Promise<void> {
  const client = getFlipClient();

  const brain = await selectBrain();
  if (!brain) {
    return;
  }

  const promptSelection = await selectPromptNote(brain);
  if (!promptSelection) {
    return;
  }

  const promptExtra = await promptExtraInstructions();
  if (promptExtra === undefined) {
    return;
  }

  const linkMode = await selectJournalLinkMode();
  if (!linkMode) {
    return;
  }

  const model = await pickModel('summarize');
  if (model === undefined) {
    return;
  }

  const topic = await vscode.window.showInputBox({
    prompt: 'Summarize topic (leave empty for interactive prompt)',
    placeHolder: 'Docker best practices',
  });

  if (topic === undefined) {
    return;
  }

  if (topic.trim() === '') {
    vscode.window.showWarningMessage('Topic is required for AI summary.');
    return;
  }

  const suggestedTitle = suggestTitleFromTopic(topic.trim());
  const title = await vscode.window.showInputBox({
    prompt: 'Title',
    placeHolder: 'Short, clear title for the note',
    value: suggestedTitle,
  });

  if (title === undefined) {
    return;
  }

  if (title.trim() === '') {
    vscode.window.showWarningMessage('Title is required for AI summary.');
    return;
  }

  const args: string[] = [
    'ai',
    'summarize',
    topic.trim(),
    '--no-stream',
    '--brain', brain,
    linkMode === 'link' ? '--link' : '--no-link',
  ];
  appendPromptArgs(args, promptSelection, promptExtra);
  if (model) {
    args.push('--model', model);
  }
  args.push('--title', title.trim());

  const result = await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Creating summary note...',
      cancellable: false,
    },
    async () => client.runCommand(args, getAiTimeoutMs())
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Failed to create summary note: ${result.error}`);
    return;
  }

  const data = result.data as NoteResult;
  const doc = await vscode.workspace.openTextDocument(data.path);
  await vscode.window.showTextDocument(doc, { preview: false });
}

export async function aiImproveCommand(): Promise<void> {
  let filePath: string | undefined;

  const editor = vscode.window.activeTextEditor;
  if (editor && editor.document && !editor.document.isUntitled) {
    filePath = editor.document.uri.fsPath;
  }

  if (!filePath) {
    const picked = await vscode.window.showOpenDialog({
      canSelectFiles: true,
      canSelectFolders: false,
      canSelectMany: false,
      openLabel: 'Select note to improve',
    });
    if (!picked || picked.length === 0) {
      return;
    }
    filePath = picked[0].fsPath;
  }

  const client = getFlipClient();
  let promptSelection: PromptSelection | undefined;
  const info = await client.getInfo();
  if (info.success && info.data) {
    const matchedBrain = findBrainForFile(filePath, info.data.brains);
    if (matchedBrain) {
      promptSelection = await selectPromptNote(matchedBrain.name);
      if (!promptSelection) {
        return;
      }
    }
  }

  const promptExtra = await promptExtraInstructions();
  if (promptExtra === undefined) {
    return;
  }

  const model = await pickModel('improve');
  if (model === undefined) {
    return;
  }

  const instruction = await vscode.window.showInputBox({
    prompt: 'How should the note be improved?',
    value: 'improve clarity and structure',
  });

  if (instruction === undefined) {
    return;
  }

  const mode = await vscode.window.showQuickPick(
    [
      { label: 'Apply changes in place', value: 'in-place' },
      { label: 'Preview in new document', value: 'preview' },
    ],
    { placeHolder: 'How should the improved note be applied?' }
  );

  if (!mode) {
    return;
  }

  const args: string[] = ['ai', 'improve', filePath, '--no-stream'];
  if (promptSelection) {
    appendPromptArgs(args, promptSelection, promptExtra);
  } else if (promptExtra.trim() !== '') {
    args.push('--prompt', 'none');
    args.push('--prompt-extra', promptExtra.trim());
  }
  if (instruction.trim() !== '') {
    args.push('--instruction', instruction.trim());
  }
  if (model) {
    args.push('--model', model);
  }
  if (mode.value === 'in-place') {
    args.push('--in-place');
  }

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Improving note with AI...',
      cancellable: false,
    },
    async () => {
      try {
        const execPath = getFlipExecutablePath();
        const result = await execFileAsync(execPath, args, {
          timeout: getAiTimeoutMs(),
          env: { ...process.env, TERM_PROGRAM: 'vscode' },
        });

        const output = (result.stdout || '').trim();

        if (mode.value === 'in-place') {
          const doc = await vscode.workspace.openTextDocument(filePath!);
          await vscode.window.showTextDocument(doc, { preview: false });
          vscode.window.showInformationMessage('Note updated in place.');
          return;
        }

        if (output === '') {
          vscode.window.showWarningMessage('No output received from AI.');
          return;
        }

        const previewDoc = await vscode.workspace.openTextDocument({
          language: 'markdown',
          content: output,
        });
        await vscode.window.showTextDocument(previewDoc, {
          preview: true,
          viewColumn: vscode.ViewColumn.Beside,
        });
      } catch (error: any) {
        const message = error?.stderr?.toString?.() || error?.message || 'AI improve failed';
        vscode.window.showErrorMessage(message);
      }
    }
  );
}
