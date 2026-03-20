import * as vscode from 'vscode';
import * as path from 'path';
import * as os from 'os';
import * as fs from 'fs/promises';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { getFlipClient, BrainInfo, getFlipExecutablePath, NoteResult, PromptsResult } from '../flip-client';
import { isCopilotAvailable, getCopilotModels, sendCopilotRequestWithProgress, CopilotModel } from './copilot-provider';

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

let aiTerminal: vscode.Terminal | undefined;

function getAiTerminal(): vscode.Terminal {
  if (!aiTerminal || aiTerminal.exitStatus !== undefined) {
    aiTerminal = vscode.window.createTerminal('Flip AI');
  }
  return aiTerminal;
}

export function disposeAiTerminal(): void {
  aiTerminal?.dispose();
  aiTerminal = undefined;
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

  // Fetch CLI provider models and Copilot models in parallel
  const [cliResult, copilotModels] = await Promise.all([
    client.runCommand(['ai', 'models']).catch(() => ({ success: false, data: undefined, error: 'failed' })),
    getCopilotModels().catch(() => [] as CopilotModel[]),
  ]);

  const data = (cliResult.success && cliResult.data) ? cliResult.data as AIModelsResponse : undefined;
  const models = data?.models || [];

  // If neither source has models, warn and return undefined (cancel)
  if (models.length === 0 && copilotModels.length === 0) {
    vscode.window.showWarningMessage('No AI models available. Run AI Setup or install GitHub Copilot.');
    return undefined;
  }

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

  const items: Array<vscode.QuickPickItem & { value: string }> = [];

  // Copilot models (shown first when available)
  if (copilotModels.length > 0) {
    items.push({
      label: '$(github) Copilot Models',
      kind: vscode.QuickPickItemKind.Separator,
      value: '__sep__',
    });
    for (const cm of copilotModels) {
      items.push({
        label: `$(github) ${cm.family}`,
        description: `${cm.vendor} • via Copilot`,
        value: `copilot:${cm.id}`,
      });
    }
  }

  // CLI provider models
  if (data) {
    items.push({
      label: `$(server) ${data.provider} Models`,
      kind: vscode.QuickPickItemKind.Separator,
      value: '__sep__',
    });
    items.push({
      label: `Use default (${data.provider}: ${data.default_model})`,
      description: 'recommended default',
      value: '',
    });

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
  }

  const selected = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select model (Copilot or provider)',
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
  args.push('--title', title.trim());

  // --- Copilot flow ---
  if (isCopilotModel(model)) {
    const copilotModelId = extractCopilotModelId(model);
    try {
      await runCopilotFlow('research', args, copilotModelId);
    } catch (err: any) {
      vscode.window.showErrorMessage(`Copilot research failed: ${err.message || err}`);
    }
    return;
  }

  // --- Standard CLI flow ---
  if (model) {
    args.push('--model', model);
  }
  
  

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
  args.push('--title', title.trim());

  // --- Copilot flow ---
  if (isCopilotModel(model)) {
    const copilotModelId = extractCopilotModelId(model);
    try {
      await runCopilotFlow('summarize', args, copilotModelId);
    } catch (err: any) {
      vscode.window.showErrorMessage(`Copilot summarize failed: ${err.message || err}`);
    }
    return;
  }

  // --- Standard CLI flow ---
  if (model) {
    args.push('--model', model);
  }
  

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
  if (mode.value === 'in-place') {
    args.push('--in-place');
  }

  // --- Copilot flow ---
  if (isCopilotModel(model)) {
    const copilotModelId = extractCopilotModelId(model);
    try {
      await runCopilotImproveFlow(args, copilotModelId, filePath!, mode.value === 'in-place');
    } catch (err: any) {
      vscode.window.showErrorMessage(`Copilot improve failed: ${err.message || err}`);
    }
    return;
  }

  // --- Standard CLI flow ---
  if (model) {
    args.push('--model', model);
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


/** Check whether the selected model value is a Copilot model (prefixed with "copilot:"). */
function isCopilotModel(model: string | null): model is string {
  return typeof model === 'string' && model.startsWith('copilot:');
}

/** Extract the actual model ID from a "copilot:xxx" value. */
function extractCopilotModelId(model: string): string {
  return model.replace(/^copilot:/, '');
}

/** Parsed result from flip CLI --prepare-only. */
interface PrepareOnlyResult {
  system_prompt: string;
  user_prompt: string;
  output_path: string;
  frontmatter: string;
  prompt_section?: string;
  sources_section?: string;
  brain_name?: string;
  brain_path?: string;
  title?: string;
  topic?: string;
  file_path?: string;
  in_place?: boolean;
  instruction?: string;
}

/**
 * Run the full Copilot flow for research/summarize:
 * 1. Call CLI with --prepare-only to get prompts
 * 2. Send to Copilot via vscode.lm
 * 3. Write the result file
 * 4. Open in editor
 */
async function runCopilotFlow(
  action: 'research' | 'summarize',
  cliArgs: string[],
  copilotModelId: string,
): Promise<void> {
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Creating ${action} note via Copilot…`,
      cancellable: true,
    },
    async (progress, token) => {
      // Step 1: Get prompts from CLI
      progress.report({ message: 'Preparing prompts…' });
      const prepArgs = [...cliArgs, '--prepare-only'];
      const execPath = getFlipExecutablePath();
      const result = await promisify(execFile)(execPath, prepArgs, {
        timeout: 30000,
        env: { ...process.env, TERM_PROGRAM: 'vscode' },
      });

      const stdout = (result.stdout || '').trim();
      let prepared: PrepareOnlyResult;
      try {
        prepared = JSON.parse(stdout);
      } catch {
        throw new Error(`Failed to parse prepare-only output: ${stdout.substring(0, 200)}`);
      }

      if (token.isCancellationRequested) {
        return;
      }

      // Step 2: Send to Copilot
      progress.report({ message: `Sending to Copilot (${copilotModelId})…` });
      const aiResponse = await sendCopilotRequestWithProgress(
        prepared.system_prompt,
        prepared.user_prompt,
        copilotModelId,
        progress,
        token,
      );

      if (token.isCancellationRequested) {
        return;
      }

      // Step 3: Build and write the final file
      progress.report({ message: 'Writing note…' });
      const frontmatter = prepared.frontmatter
        .replace('__PROVIDER__', 'copilot')
        .replace('__MODEL__', copilotModelId);
      const promptSection = prepared.prompt_section || '';
      const sourcesSection = prepared.sources_section || '';
      const finalContent = frontmatter + promptSection + sourcesSection + aiResponse;

      const fs = await import('fs');
      const path = await import('path');
      const dirPath = path.dirname(prepared.output_path);
      if (!fs.existsSync(dirPath)) {
        fs.mkdirSync(dirPath, { recursive: true });
      }
      fs.writeFileSync(prepared.output_path, finalContent, 'utf-8');

      // Step 4: Open in editor
      const doc = await vscode.workspace.openTextDocument(prepared.output_path);
      await vscode.window.showTextDocument(doc, { preview: false });

      vscode.window.showInformationMessage(`✨ ${action === 'research' ? 'Research' : 'Summary'} note created via Copilot!`);
    },
  );
}

/**
 * Run the Copilot flow for improve:
 * 1. Call CLI with --prepare-only to get prompts
 * 2. Send to Copilot via vscode.lm
 * 3. Apply result (in-place or preview)
 */
async function runCopilotImproveFlow(
  cliArgs: string[],
  copilotModelId: string,
  filePath: string,
  inPlace: boolean,
): Promise<void> {
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Improving note via Copilot…',
      cancellable: true,
    },
    async (progress, token) => {
      progress.report({ message: 'Preparing prompts…' });
      const prepArgs = [...cliArgs, '--prepare-only'];
      const execPath = getFlipExecutablePath();
      const result = await promisify(execFile)(execPath, prepArgs, {
        timeout: 30000,
        env: { ...process.env, TERM_PROGRAM: 'vscode' },
      });

      const stdout = (result.stdout || '').trim();
      let prepared: PrepareOnlyResult;
      try {
        prepared = JSON.parse(stdout);
      } catch {
        throw new Error(`Failed to parse prepare-only output: ${stdout.substring(0, 200)}`);
      }

      if (token.isCancellationRequested) {
        return;
      }

      progress.report({ message: `Sending to Copilot (${copilotModelId})…` });
      const aiResponse = await sendCopilotRequestWithProgress(
        prepared.system_prompt,
        prepared.user_prompt,
        copilotModelId,
        progress,
        token,
      );

      if (token.isCancellationRequested) {
        return;
      }

      if (inPlace) {
        const fs = await import('fs');
        fs.writeFileSync(filePath, aiResponse, 'utf-8');
        const doc = await vscode.workspace.openTextDocument(filePath);
        await vscode.window.showTextDocument(doc, { preview: false });
        vscode.window.showInformationMessage('Note updated in place via Copilot.');
      } else {
        const previewDoc = await vscode.workspace.openTextDocument({
          language: 'markdown',
          content: aiResponse,
        });
        await vscode.window.showTextDocument(previewDoc, {
          preview: true,
          viewColumn: vscode.ViewColumn.Beside,
        });
      }
    },
  );
}
