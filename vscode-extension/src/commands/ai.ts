import * as vscode from 'vscode';
import * as path from 'path';
import * as os from 'os';
import * as fs from 'fs/promises';
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

type AISetupProvider = 'ollama' | 'openai' | 'anthropic' | 'groq' | 'mistral' | 'azure';

interface PromptSelection {
  mode: 'none' | 'existing' | 'create';
  value?: string;
  title?: string;
  body?: string;
  setDefault?: boolean;
}

interface TemplatePreset {
  key: 'research' | 'summarize' | 'improve' | 'blank';
  title: string;
  body: string;
}

interface ContextSelection {
  files: string[];
  autoBrain: boolean;
}

interface RunReviewConfig {
  action: 'research' | 'summarize' | 'improve';
  brain?: string;
  template: PromptSelection;
  request?: string;
  instruction?: string;
  runNotes: string;
  context: ContextSelection;
  model: string | null;
  title?: string;
  targetFile?: string;
  applyMode?: 'in-place' | 'preview';
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

function defaultPromptEditorContent(title: string, initialContent: string): string {
  if (initialContent.trim() !== '') {
    return initialContent;
  }

  if (title === 'Template instructions') {
    return [
      '# Template instructions',
      '',
      '## Rolle',
      '',
      'Du bist ein erfahrener Assistent für ...',
      '',
      '## Aufgabe',
      '',
      '- Analysiere ...',
      '- Erstelle ...',
      '',
      '## Kontext',
      '',
      '- Ziel:',
      '- Zielgruppe:',
      '- Wichtige Rahmenbedingungen:',
      '',
      '## Ausgabe-Format',
      '',
      '- Sprache: Deutsch',
      '- Stil: klar, präzise, pragmatisch',
      '- Struktur: gut gegliedert',
      '',
      '## Einschränkungen',
      '',
      '- ',
      '',
    ].join('\n');
  }

  if (title === 'Run notes') {
    return [
      '# Run notes',
      '',
      '- Fokus:',
      '- Stil:',
      '- Format:',
      '- Wichtig:',
      '',
    ].join('\n');
  }

  return initialContent;
}

async function promptMultiLineInput(title: string, initialContent: string): Promise<string | undefined> {
  const seedContent = defaultPromptEditorContent(title, initialContent);
  const safeName = title.toLowerCase().replace(/[^a-z0-9]+/g, '-');
  const tempDir = path.join(os.tmpdir(), 'flip-ai-input');
  await fs.mkdir(tempDir, { recursive: true });
  const tempPath = path.join(tempDir, `${safeName}-${Date.now()}.md`);
  await fs.writeFile(tempPath, seedContent, 'utf8');

  const uri = vscode.Uri.file(tempPath);
  const doc = await vscode.workspace.openTextDocument(uri);
  await vscode.window.showTextDocument(doc, { preview: false });

  const choice = await vscode.window.showInformationMessage(
    title,
    { detail: 'Edit and save the temporary file, then choose “Use content”.' },
    'Use content',
    'Cancel'
  );

  if (choice !== 'Use content') {
    return undefined;
  }

  if (doc.isDirty) {
    await doc.save();
  }

  const content = await fs.readFile(tempPath, 'utf8');
  return content;
}

async function promptFileInput(title: string, filePath: string, initialContent: string): Promise<string | undefined> {
  await fs.mkdir(path.dirname(filePath), { recursive: true });

  try {
    await fs.access(filePath);
  } catch {
    await fs.writeFile(filePath, initialContent, 'utf8');
  }

  const uri = vscode.Uri.file(filePath);
  const doc = await vscode.workspace.openTextDocument(uri);
  await vscode.window.showTextDocument(doc, { preview: false });

  const choice = await vscode.window.showInformationMessage(
    title,
    { detail: `Edit and save the template file, then choose “Use content”.\n${filePath}` },
    'Use content',
    'Cancel'
  );

  if (choice !== 'Use content') {
    return undefined;
  }

  if (doc.isDirty) {
    await doc.save();
  }

  return fs.readFile(filePath, 'utf8');
}

async function promptRunNotes(): Promise<string | undefined> {
  const choice = await vscode.window.showQuickPick(
    [
      { label: 'No run notes', value: 'none' },
      { label: 'Add run notes (editor)', value: 'editor' },
    ],
    { placeHolder: 'Add optional run notes for this request?' }
  );

  if (!choice) {
    return undefined;
  }

  if (choice.value === 'none') {
    return '';
  }

  const content = await promptMultiLineInput('Run notes', '');
  if (content === undefined) {
    return undefined;
  }

  return content.trim();
}

function appendPromptArgs(args: string[], selection: PromptSelection, extra: string | undefined): void {
  if (selection.mode === 'existing' && selection.value) {
    args.push('--template', selection.value);
  } else if (selection.mode === 'create' && selection.title) {
    args.push('--template-create-title', selection.title);
    args.push('--template-create-body', selection.body || '');
    if (selection.setDefault) {
      args.push('--template-create-default');
    }
  } else {
    args.push('--template', 'none');
  }

  if (extra && extra.trim() !== '') {
    args.push('--run-notes', extra.trim());
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

function formatTemplateSelection(selection: PromptSelection): string {
  if (selection.mode === 'none') {
    return 'None';
  }
  if (selection.mode === 'create') {
    return `Create: ${selection.title || 'Untitled'}`;
  }
  if (selection.mode === 'existing') {
    const value = selection.value || '';
    if (!value) {
      return 'Default';
    }
    return path.basename(value);
  }
  return 'None';
}

function formatContextSelection(context: ContextSelection): string {
  const parts: string[] = [];
  if (context.autoBrain) {
    parts.push('Auto brain search');
  }
  if (context.files.length > 0) {
    if (context.files.length === 1) {
      parts.push(`1 file: ${path.basename(context.files[0])}`);
    } else {
      parts.push(`${context.files.length} files`);
    }
  }
  if (parts.length === 0) {
    return 'No additional context';
  }
  return parts.join(' + ');
}

function getTemplatePresets(action: 'research' | 'summarize' | 'improve'): TemplatePreset[] {
  const researchPreset: TemplatePreset = {
    key: 'research',
    title: 'Research (Best Practice)',
    body: [
      '## Role',
      '',
      'You are a senior research analyst producing decision-ready insights.',
      '',
      '## Objective',
      '',
      '- Build a structured overview of the request.',
      '- Separate facts, assumptions, and open questions.',
      '- Highlight practical recommendations and risks.',
      '',
      '## Output Format',
      '',
      '- Executive summary (3-6 bullets)',
      '- Key findings',
      '- Options with trade-offs',
      '- Recommended next steps',
      '',
      '## Quality Rules',
      '',
      '- Be concise, concrete, and action-oriented.',
      '- Call out uncertainty explicitly.',
      '- Avoid generic advice.',
      '',
    ].join('\n'),
  };

  const summarizePreset: TemplatePreset = {
    key: 'summarize',
    title: 'Summarize (Best Practice)',
    body: [
      '## Role',
      '',
      'You are an editor creating high-signal summaries from source notes.',
      '',
      '## Objective',
      '',
      '- Distill the core message without losing important nuance.',
      '- Keep important entities, dates, decisions, and dependencies.',
      '- Remove repetition and low-value details.',
      '',
      '## Output Format',
      '',
      '- TL;DR (2-4 bullets)',
      '- Main points',
      '- Decisions and open questions',
      '- Follow-up actions',
      '',
      '## Quality Rules',
      '',
      '- Keep wording clear and neutral.',
      '- Do not invent missing details.',
      '- Prefer bullet points over long paragraphs.',
      '',
    ].join('\n'),
  };

  const improvePreset: TemplatePreset = {
    key: 'improve',
    title: 'Improve (Best Practice)',
    body: [
      '## Role',
      '',
      'You are a technical writer improving clarity and structure of existing notes.',
      '',
      '## Objective',
      '',
      '- Keep meaning intact while improving readability.',
      '- Strengthen structure, headings, and flow.',
      '- Make action items and decisions easier to scan.',
      '',
      '## Output Format',
      '',
      '- Return improved markdown only.',
      '- Preserve frontmatter and important links.',
      '',
      '## Quality Rules',
      '',
      '- Do not remove critical context.',
      '- Keep terminology consistent with the original note.',
      '- Prefer precise wording over stylistic flourishes.',
      '',
    ].join('\n'),
  };

  const blankPreset: TemplatePreset = {
    key: 'blank',
    title: '',
    body: '',
  };

  if (action === 'research') {
    return [researchPreset, summarizePreset, improvePreset, blankPreset];
  }
  if (action === 'summarize') {
    return [summarizePreset, researchPreset, improvePreset, blankPreset];
  }
  return [improvePreset, summarizePreset, researchPreset, blankPreset];
}

function slugifyTemplateTitle(title: string): string {
  const slug = title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return slug || 'template';
}

function buildTemplateFileContent(title: string, body: string, setDefault: boolean): string {
  const date = new Date().toISOString().slice(0, 10);
  const lines = [
    '---',
    `title: ${title}`,
    `created: ${date}`,
    'type: prompt',
    'tags: [prompt]',
  ];

  if (setDefault) {
    lines.push('prompt_default: true');
  }

  lines.push('---', '', `# ${title}`, '');

  const trimmed = body.trim();
  if (trimmed !== '') {
    lines.push(trimmed, '');
  }

  return lines.join('\n');
}

async function promptTemplateCreation(action: 'research' | 'summarize' | 'improve', brainPath: string): Promise<PromptSelection | undefined> {
  const presets = getTemplatePresets(action);
  const presetChoice = await vscode.window.showQuickPick(
    [
      { label: 'Research template (best practice)', value: 'research' },
      { label: 'Summarize template (best practice)', value: 'summarize' },
      { label: 'Improve template (best practice)', value: 'improve' },
      { label: 'Blank custom template', value: 'blank' },
    ],
    { placeHolder: 'Choose a template preset' }
  );

  if (!presetChoice) {
    return undefined;
  }

  const selectedPreset = presets.find((preset) => preset.key === presetChoice.value) || presets[0];

  const title = await vscode.window.showInputBox({
    prompt: 'Template title',
    placeHolder: selectedPreset.title || 'Research (Deep)',
    value: selectedPreset.title,
  });
  if (title === undefined) {
    return undefined;
  }
  if (title.trim() === '') {
    vscode.window.showWarningMessage('Template title is required.');
    return { mode: 'none' };
  }

  const setDefaultChoice = await vscode.window.showQuickPick(
    [
      { label: 'Set as default template', value: 'yes' },
      { label: 'Do not set default', value: 'no' },
    ],
    { placeHolder: 'Default template?' }
  );
  if (!setDefaultChoice) {
    return undefined;
  }

  const templateDir = path.join(brainPath, 'templates', 'ai');
  const fileName = `${slugifyTemplateTitle(title.trim())}.md`;
  const templatePath = path.join(templateDir, fileName);

  const initialContent = buildTemplateFileContent(
    title.trim(),
    selectedPreset.body,
    setDefaultChoice.value === 'yes'
  );

  const content = await promptFileInput('Template instructions', templatePath, initialContent);
  if (content === undefined) {
    return undefined;
  }

  return {
    mode: 'existing',
    value: templatePath,
  };
}

function truncateForReview(value: string, max = 140): string {
  const normalized = value.replace(/\s+/g, ' ').trim();
  if (normalized.length <= max) {
    return normalized;
  }
  return normalized.slice(0, max - 1) + '…';
}

async function confirmRunReview_UNUSED(config: RunReviewConfig): Promise<boolean> {
  const lines: string[] = [];
  if (config.brain) {
    lines.push(`Brain: ${config.brain}`);
  }
  lines.push(`Template: ${formatTemplateSelection(config.template)}`);

  if (config.request && config.request.trim() !== '') {
    lines.push(`Request: ${truncateForReview(config.request)}`);
  }

  if (config.instruction && config.instruction.trim() !== '') {
    lines.push(`Instruction: ${truncateForReview(config.instruction)}`);
  }

  lines.push(`Context: ${formatContextSelection(config.context)}`);
  lines.push(`Run notes: ${config.runNotes.trim() ? 'Yes' : 'No'}`);
  lines.push(`Model: ${config.model || 'Provider default'}`);

  if (config.title) {
    lines.push(`Title: ${config.title}`);
  }
  if (config.targetFile) {
    lines.push(`File: ${path.basename(config.targetFile)}`);
  }
  if (config.applyMode) {
    lines.push(`Apply mode: ${config.applyMode}`);
  }

  const detail = lines.join('\n');
  const actionLabel = config.action === 'improve' ? 'Run improve' : `Run ${config.action}`;
  const choice = await vscode.window.showInformationMessage(
    'Review AI run configuration',
    { modal: true, detail },
    actionLabel,
    'Cancel'
  );

  return choice === actionLabel;
}

async function selectPromptNote(brain: string, action: 'research' | 'summarize' | 'improve'): Promise<PromptSelection | undefined> {
  const client = getFlipClient();
  const result = await client.runCommand(['vscode', 'prompts', '--brain', brain]);

  if (!result.success || !result.data) {
    return { mode: 'none' };
  }

  const data = result.data as PromptsResult;
  const prompts = data.prompts || [];
  const brainPath = data.brain_path || '';

  if (!brainPath) {
    vscode.window.showErrorMessage('Could not resolve brain path for template management.');
    return undefined;
  }

  // Disable auto-return to let the user select variants!
  // if (data.default_prompt) {
  //   return { mode: 'existing', value: data.default_prompt };
  // }

  if (prompts.length === 0) {
    const choice = await vscode.window.showQuickPick(
      [
        { label: 'Create new template', value: 'create' },
        { label: 'No template', value: 'none' },
      ],
      { placeHolder: 'No template notes found' }
    );

    if (!choice) {
      return undefined;
    }

    if (choice.value === 'none') {
      return { mode: 'none' };
    }
    return promptTemplateCreation(action, brainPath);
  }

  const items: Array<vscode.QuickPickItem & { value: string }> = [
    { label: 'Create new template', value: 'create' },
    { label: 'No template', value: 'none' },
  ];

  // Move default prompt to top and mark it
  if (data.default_prompt) {
    const defaultPromptObj = prompts.find(p => p.path === data.default_prompt);
    if (defaultPromptObj) {
      items.push({
        label: `$(star) ${defaultPromptObj.title || defaultPromptObj.name} (Default)`,
        description: defaultPromptObj.rel_path,
        value: defaultPromptObj.path,
        picked: true
      });
    }
  }

  for (const prompt of prompts) {
    if (prompt.path === data.default_prompt) continue;
    items.push({
      label: prompt.title || prompt.name,
      description: prompt.rel_path,
      value: prompt.path,
    });
  }

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select template note (optional)',
  });

  if (!selected) {
    return undefined;
  }

  if (selected.value === 'create') {
    return promptTemplateCreation(action, brainPath);
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

function getDefaultSetupModel(provider: AISetupProvider): string {
  switch (provider) {
    case 'openai':
      return 'gpt-4o-mini';
    case 'anthropic':
      return 'claude-3-5-haiku-20241022';
    case 'groq':
      return 'llama-3.3-70b-versatile';
    case 'mistral':
      return 'mistral-large-latest';
    case 'azure':
      return 'my-azure-deployment';
    default:
      return 'llama3.2';
  }
}

function getSetupModelPlaceholder(provider: AISetupProvider): string {
  switch (provider) {
    case 'openai':
      return 'e.g. gpt-4o-mini';
    case 'anthropic':
      return 'e.g. claude-3-5-haiku-20241022';
    case 'groq':
      return 'e.g. llama-3.3-70b-versatile';
    case 'mistral':
      return 'e.g. mistral-large-latest, mistral-small-latest';
    case 'azure':
      return 'Azure deployment name';
    default:
      return 'e.g. llama3.2, mistral, gemma3';
  }
}

function parseModelsResponse(output: string): AIModelsResponse | undefined {
  const trimmed = output.trim();
  if (!trimmed) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (parsed?.data?.models && parsed?.data?.provider) {
      return parsed.data as AIModelsResponse;
    }
    if (parsed?.models && parsed?.provider) {
      return parsed as AIModelsResponse;
    }
  } catch {
    return undefined;
  }

  return undefined;
}

async function fetchModelsForProvider(provider: AISetupProvider): Promise<AIModelsResponse | undefined> {
  try {
    const execPath = getFlipExecutablePath();
    const result = await execFileAsync(execPath, ['ai', 'models', '--json'], {
      timeout: 300000 /* 5 minutes */,
      env: {
        ...process.env,
        TERM_PROGRAM: 'vscode',
        FLIP_AI_PROVIDER: provider,
      },
    });

    return parseModelsResponse(result.stdout || '');
  } catch {
    return undefined;
  }
}

async function pickSetupProvider(): Promise<AISetupProvider | undefined> {
  const selected = await vscode.window.showQuickPick(
    [
      { label: 'Ollama (local)', value: 'ollama' as AISetupProvider, detail: 'Local LLM, no API key required' },
      { label: 'OpenAI', value: 'openai' as AISetupProvider, detail: 'Requires OPENAI_API_KEY' },
      { label: 'Anthropic', value: 'anthropic' as AISetupProvider, detail: 'Requires ANTHROPIC_API_KEY' },
      { label: 'Groq', value: 'groq' as AISetupProvider, detail: 'Requires GROQ_API_KEY' },
      { label: 'Mistral', value: 'mistral' as AISetupProvider, detail: 'Requires MISTRAL_API_KEY' },
      { label: 'Azure OpenAI', value: 'azure' as AISetupProvider, detail: 'Requires Azure endpoint + key' },
    ],
    { placeHolder: 'Select AI provider to configure' }
  );

  return selected?.value;
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

async function selectContextSelection(action: 'research' | 'summarize'): Promise<ContextSelection | undefined> {
  const options: Array<vscode.QuickPickItem & { value: string }> = [
    { label: 'No additional context', value: 'none' },
    { label: 'Use current file as context', value: 'current' },
    { label: 'Select file(s) as context', value: 'select' },
  ];

  if (action === 'research') {
    options.push({ label: 'Auto-search current brain context', value: 'auto-brain' });
  }

  const choice = await vscode.window.showQuickPick(options, {
    placeHolder: 'Select context source (optional)',
  });

  if (!choice) {
    return undefined;
  }

  if (choice.value === 'none') {
    return { files: [], autoBrain: false };
  }

  if (choice.value === 'auto-brain') {
    return { files: [], autoBrain: true };
  }

  if (choice.value === 'current') {
    const activeDoc = vscode.window.activeTextEditor?.document;
    if (!activeDoc || activeDoc.isUntitled) {
      vscode.window.showWarningMessage('No saved active file available as context.');
      return { files: [], autoBrain: false };
    }
    return { files: [activeDoc.uri.fsPath], autoBrain: false };
  }

  const picked = await vscode.window.showOpenDialog({
    canSelectFiles: true,
    canSelectFolders: false,
    canSelectMany: true,
    openLabel: 'Use selected files as context',
  });

  if (!picked || picked.length === 0) {
    return { files: [], autoBrain: false };
  }

  return { files: picked.map((item) => item.fsPath), autoBrain: false };
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

  const promptSelection = await selectPromptNote(brain, 'research');
  if (!promptSelection) {
    return;
  }

  const request = await vscode.window.showInputBox({
    prompt: 'Research request (you can paste multiple sentences here)',
    placeHolder: 'Analyze Kubernetes networking trade-offs for production clusters',
  });

  if (request === undefined) {
    return;
  }

  if (request.trim() === '') {
    vscode.window.showWarningMessage('Request is required for AI research.');
    return;
  }

  const contextSelection = await selectContextSelection('research');
  if (!contextSelection) {
    return;
  }

  const promptExtra = ""; // Skipped to reduce clicks

  const linkMode = await selectJournalLinkMode();
  if (!linkMode) {
    return;
  }

  const model = await pickModel('research');
  if (model === undefined) {
    return;
  }

  const suggestedTitle = suggestTitleFromTopic(request.trim());
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

  

  
  // PRE-CREATE AND OPEN FILE
  const info = await client.getInfo();

  const args: string[] = [
    'ai',
    'research',
    '--request', request.trim(),
    '--no-stream',
    '--brain', brain,
    linkMode === 'link' ? '--link' : '--no-link',
  ];
  appendPromptArgs(args, promptSelection, promptExtra);
  for (const file of contextSelection.files) {
    args.push('--context-file', file);
  }
  if (contextSelection.autoBrain) {
    args.push('--context-auto-brain');
  }
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
    async () => client.runCommand(args, 300000 /* 5 minutes */)
  );

  if (!result.success || !result.data) {
    if (result.error && result.error.includes("AI_SETUP_REQUIRED")) {
      vscode.commands.executeCommand('flip.aiSetup');
      return;
    }
    vscode.window.showErrorMessage(`Failed to create research note: ${result.error}`);
    return;
  }

  vscode.window.showInformationMessage("✨ AI note generated successfully!");
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

  const promptSelection = await selectPromptNote(brain, 'summarize');
  if (!promptSelection) {
    return;
  }

  const request = await vscode.window.showInputBox({
    prompt: 'Summarize request',
    placeHolder: 'Summarize project Phoenix decisions and open risks',
  });

  if (request === undefined) {
    return;
  }

  if (request.trim() === '') {
    vscode.window.showWarningMessage('Request is required for AI summary.');
    return;
  }

  const contextSelection = await selectContextSelection('summarize');
  if (!contextSelection) {
    return;
  }

  const promptExtra = ""; // Skipped to reduce clicks

  const linkMode = await selectJournalLinkMode();
  if (!linkMode) {
    return;
  }

  const model = await pickModel('summarize');
  if (model === undefined) {
    return;
  }

  const suggestedTitle = suggestTitleFromTopic(request.trim());
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

  

  
  // PRE-CREATE AND OPEN FILE
  const info = await client.getInfo();

  const args: string[] = [
    'ai',
    'summarize',
    '--request', request.trim(),
    '--no-stream',
    '--brain', brain,
    linkMode === 'link' ? '--link' : '--no-link',
  ];
  appendPromptArgs(args, promptSelection, promptExtra);
  for (const file of contextSelection.files) {
    args.push('--context-file', file);
  }
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
    async () => client.runCommand(args, 300000 /* 5 minutes */)
  );

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Failed to create summary note: ${result.error}`);
    return;
  }

  vscode.window.showInformationMessage("✨ AI note generated successfully!");
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
      promptSelection = await selectPromptNote(matchedBrain.name, 'improve');
      if (!promptSelection) {
        return;
      }
    }
  }

  const promptExtra = ""; // Skipped to reduce clicks

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
    args.push('--template', 'none');
    args.push('--run-notes', promptExtra.trim());
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
          timeout: 300000 /* 5 minutes */,
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

export async function aiSetupCommand(): Promise<void> {
  const provider = await pickSetupProvider();
  if (!provider) {
    return;
  }

  const modelsData = await fetchModelsForProvider(provider);
  let model = '';

  if (modelsData && modelsData.models && modelsData.models.length > 0) {
    const items: Array<vscode.QuickPickItem & { value: string }> = [
      {
        label: `Use provider default (${modelsData.default_model || getDefaultSetupModel(provider)})`,
        value: '',
        description: 'Recommended',
      },
    ];

    for (const m of modelsData.models) {
      const id = m.id || m.name || '';
      if (!id) {
        continue;
      }
      items.push({
        label: m.name && m.name !== id ? m.name : id,
        description: m.description,
        detail: m.name && m.name !== id ? id : undefined,
        value: id,
      });
    }

    const selectedModel = await vscode.window.showQuickPick(items, {
      placeHolder: `Select model for ${provider}`,
    });

    if (!selectedModel) {
      return;
    }

    model = selectedModel.value;
  } else {
    const typedModel = await vscode.window.showInputBox({
      prompt: `Model for ${provider} (empty = provider default)`,
      value: getDefaultSetupModel(provider),
      placeHolder: getSetupModelPlaceholder(provider),
    });

    if (typedModel === undefined) {
      return;
    }

    model = typedModel.trim();
  }

  let apiKey = '';
  if (provider !== 'ollama') {
    const keyInput = await vscode.window.showInputBox({
      prompt: `Optional API key for ${provider} (used only for this setup call)`,
      placeHolder: 'Leave empty to use existing environment variables',
      password: true,
    });

    if (keyInput === undefined) {
      return;
    }

    apiKey = keyInput.trim();
  }

  let installOllama = false;
  if (provider === 'ollama') {
    const installChoice = await vscode.window.showQuickPick(
      [
        { label: 'Do not auto-install Ollama', value: 'no' },
        { label: 'Auto-install Ollama if missing', value: 'yes' },
      ],
      { placeHolder: 'If Ollama is missing, should setup install it automatically?' }
    );

    if (!installChoice) {
      return;
    }

    installOllama = installChoice.value === 'yes';
  }

  const setupArgs = ['ai', 'setup', '--provider', provider];
  if (model !== '') {
    setupArgs.push('--model', model);
  }
  if (apiKey !== '') {
    setupArgs.push('--api-key', apiKey);
  }
  if (installOllama) {
    setupArgs.push('--install-ollama');
  }

  const displayModel = model || (modelsData?.default_model || getDefaultSetupModel(provider));
  const timeout = provider === 'ollama' ? 10 * 60 * 1000 : 300000 /* 5 minutes */;

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Setting up AI (${provider}: ${displayModel})…`,
      cancellable: false,
    },
    async () => {
      try {
        const execPath = getFlipExecutablePath();
        const result = await execFileAsync(execPath, setupArgs, {
          timeout,
          env: { ...process.env, TERM_PROGRAM: 'vscode' },
        });
        const output = (result.stdout || '').trim();
        vscode.window.showInformationMessage(output || '✅ AI setup complete.');
      } catch (error: any) {
        const message = error?.stderr?.toString?.() || error?.message || 'AI setup failed';
        vscode.window.showErrorMessage(message);
      }
    }
  );
}


async function precalcAndOpenNote(client: any, brain: string, title: string, isSummary: boolean): Promise<string | undefined> {
  try {
    const info = await client.getInfo();
    if (info.success && info.data && info.data.brains) {
      const targetBrainObj = info.data.brains.find((b: any) => b.name === brain);
      if (targetBrainObj) {
        const notesDir = require('path').join(targetBrainObj.path, 'notes');
        const dateStr = new Date().toISOString().split('T')[0];
        const slug = title.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'note';
        const fsLib = require('fs');
        
        let candidate = `${slug}-${dateStr}.md`;
        let i = 1;
        while (fsLib.existsSync(require('path').join(notesDir, candidate))) {
           candidate = `${slug}-${dateStr}-${i}.md`;
           i++;
        }
        
        const fullPath = require('path').join(notesDir, candidate);
        if (!fsLib.existsSync(notesDir)) { fsLib.mkdirSync(notesDir, {recursive: true}); }
        const typeText = isSummary ? 'summary' : 'response';
        fsLib.writeFileSync(fullPath, `# ${title}\n\n_Generating AI ${typeText}..._\n`);
        
        const vscode = require('vscode');
        const doc = await vscode.workspace.openTextDocument(fullPath);
        await vscode.window.showTextDocument(doc, { preview: false });
        
        return fullPath;
      }
    }
  } catch(e) { 
    console.error("Failed to open early doc", e); 
  }
  return undefined;
}
