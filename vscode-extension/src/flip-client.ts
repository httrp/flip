import { exec, execFile } from 'child_process';
import { promisify } from 'util';
import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

const execAsync = promisify(exec);
const execFileAsync = promisify(execFile);
/**
 * Find flip executable by checking common installation locations
 * This helps when VS Code doesn't inherit the shell PATH (especially on Windows)
 */
function findFlipExecutable(): string {
  const config = vscode.workspace.getConfiguration('flip');
  const configuredPath = config.get<string>('executablePath');
  
  // If user explicitly configured a path, use it
  if (configuredPath && configuredPath !== 'flip') {
    return configuredPath;
  }

  // Common locations to check for flip binary
  const homeDir = os.homedir();
  const isWindows = process.platform === 'win32';
  const binaryName = isWindows ? 'flip.exe' : 'flip';
  
  const searchPaths = [
    // Development: find flip in workspace folders
    ...(vscode.workspace.workspaceFolders?.flatMap(folder => [
      path.join(folder.uri.fsPath, 'flip'),
      path.join(folder.uri.fsPath, 'flip', 'flip'),
    ]) || []),
    // Go bin directory (most common for go install)
    path.join(homeDir, 'go', 'bin', binaryName),
    // Local bin
    path.join(homeDir, '.local', 'bin', binaryName),
    // User bin
    path.join(homeDir, 'bin', binaryName),
    // Homebrew on macOS
    '/usr/local/bin/flip',
    '/opt/homebrew/bin/flip',
    // Linux system paths
    '/usr/bin/flip',
    '/usr/local/bin/flip',
  ];

  // Check each path
  for (const searchPath of searchPaths) {
    try {
      if (fs.existsSync(searchPath)) {
        console.log(`[Flip] Found flip at: ${searchPath}`);
        return searchPath;
      }
    } catch {
      // Ignore errors, continue searching
    }
  }

  // Fall back to 'flip' and hope it's in PATH
  console.log('[Flip] Using default: flip (from PATH)');
  return 'flip';
}

// Expose the resolved executable path for non-JSON commands.
export function getFlipExecutablePath(): string {
  return findFlipExecutable();
}

/**
 * Result from flip CLI command
 */
export interface FlipResult<T> {
  success: boolean;
  command: string;
  data?: T;
  error?: string;
}

/**
 * Brain information
 */
export interface BrainInfo {
  name: string;
  path: string;
  type: string;
  active: boolean;
  git_branch?: string;
  git_status?: string;
}

/**
 * Info response from flip vscode info
 */
export interface FlipInfo {
  flip_version: string;
  active_workspace: string;
  active_brain: string;
  brains: BrainInfo[];
  commands: string[];
}

/**
 * Journal creation result
 */
export interface JournalResult {
  action: 'created' | 'opened';
  path: string;
  date: string;
  brain_name: string;
  brain_path: string;
  brain_type: string;
}

/**
 * Note creation result
 */
export interface NoteResult {
  action: string;
  path: string;
  title: string;
  tags?: string[];
  brain_name: string;
  brain_path: string;
  brain_type: string;
}

export interface PromptNoteInfo {
  name: string;
  title: string;
  path: string;
  rel_path: string;
  is_default: boolean;
  brain_name: string;
  brain_type: string;
}

export interface PromptsResult {
  prompts: PromptNoteInfo[];
  brain_name: string;
  brain_type: string;
  brain_path: string;
  default_prompt?: string;
}

/**
 * Meeting series info
 */
export interface MeetingSeriesInfo {
  name: string;
  count: number;
}

/**
 * Meeting series list result
 */
export interface MeetingSeriesResult {
  series: MeetingSeriesInfo[];
  brain_name: string;
  brain_path: string;
}

/**
 * Meeting organizations list result (from scanned meetings - kept for audit)
 */
export interface MeetingOrganizationInfo {
  name: string;
  count: number;
}

export interface MeetingOrganizationsResult {
  organizations: MeetingOrganizationInfo[];
  brain_name: string;
  brain_path: string;
}

/**
 * Definition item from definitions system
 */
export interface DefinitionItem {
  name: string;
  abbreviation: string;
  description?: string;
  organization?: string;
}

/**
 * Definitions list result
 */
export interface DefinitionsResult {
  organizations: DefinitionItem[];
  projects: DefinitionItem[];
  contexts: DefinitionItem[];
  people: DefinitionItem[];
  source: string;
}

/**
 * Missing journal entry
 */
export interface MissingJournalEntry {
  path: string;
  rel_path: string;
  title: string;
  type: string;
  modified: string;
  date: string;
}

/**
 * Day result for journal sync
 */
export interface DayResult {
  date: string;
  journal_path: string;
  journal_exists: boolean;
  missing_entries: MissingJournalEntry[];
  already_linked: number;
  ignored: number;
}

/**
 * Journal sync check result
 */
export interface JournalSyncResult {
  days: DayResult[];
  total_missing: number;
  total_checked: number;
  total_linked: number;
  total_ignored: number;
  days_checked: number;
  brain_name: string;
  brain_path: string;
}

/**
 * Task creation result
 */
export interface TaskResult {
  action: string;
  path: string;
  line?: number;
  description: string;
  due?: string;
  priority?: string;
  frog?: boolean;
  status?: string;
  brain_name: string;
  brain_path: string;
}

/**
 * Task info from vscode tasks command
 */
export interface TaskInfo {
  description: string;
  status: 'open' | 'in-progress' | 'done';
  priority?: string;
  due?: string;
  tags?: string[];
  frog?: boolean;
  path: string;
  rel_path: string;
  line: number;
  brain_name: string;
  brain_type: string;
  organization?: string;
  project?: string;
  context?: string;
}

/**
 * Tasks list result
 */
export interface TasksResult {
  tasks: TaskInfo[];
  total_count: number;
  brain_name?: string;
  brain_path?: string;
  query?: string;
}

/**
 * Note info for linking
 */
export interface NoteInfo {
  name: string;
  path: string;
  rel_path: string;
  link_format: string;
  brain_name: string;
  brain_type: string;
}

/**
 * Notes list result
 */
export interface NotesResult {
  notes: NoteInfo[];
  brain_name: string;
  brain_type: string;
  brain_path: string;
  link_syntax: string;
}

/**
 * Status result
 */
export interface StatusResult {
  workspace: { name: string; path: string; active: boolean };
  active_brain?: BrainInfo;
  brains: BrainInfo[];
  flip_version: string;
  is_vscode: boolean;
}

/**
 * Search result item from vscode search/recent
 */
export interface SearchItem {
  title: string;
  path: string;
  rel_path: string;
  brain_name: string;
  brain_type: string;
  type: 'note' | 'journal' | 'meeting' | 'task';
  tags?: string[];
  modified: string;
  match_line?: number;
  match_text?: string;
}

/**
 * Search results from vscode search/recent commands
 */
export interface SearchResults {
  results: SearchItem[];
  total_count: number;
  query?: string;
  search_type: 'fulltext' | 'recent';
}

// Brain health results (JSON output from `flip brain check health --json`)
export interface HealthBrainInfo {
  Path: string;
  Type: string;
  Name: string;
  NoteCount: number;
  AssetCount: number;
}

export interface HealthStats {
  FilesScanned: number;
  LinksChecked: number;
  AssetsChecked: number;
  ErrorCount: number;
  WarningCount: number;
  InfoCount: number;
}

export interface HealthIssue {
  Type: string;
  Severity: string;
  File: string;
  Line: number;
  Message: string;
  Details: string;
}

export interface HealthResult {
  BrainInfo: HealthBrainInfo;
  Issues: HealthIssue[];
  Stats: HealthStats;
}

export interface HealthRepairStats {
  TotalIssues: number;
  Repaired: number;
  Failed: number;
  Skipped: number;
  NotRepairable: number;
}

export interface HealthRepairResult {
  Issue: HealthIssue;
  Success: boolean;
  Message: string;
  SkipReason: string;
}

export interface HealthRepairPayload {
  results: HealthRepairResult[];
  stats: HealthRepairStats;
  not_repairable_count: number;
}

export interface HealthReport {
  result: HealthResult;
  repairs?: HealthRepairPayload;
}

export interface ExportTemplate {
  name: string;
  path: string;
  source: string;
  kind: 'latex' | 'html' | 'docx';
}

export interface ExportTemplateListResult {
  format: 'pdf' | 'html' | 'docx';
  count: number;
  templates: ExportTemplate[];
}

export interface ExportConvertResult {
  input: string;
  output: string;
  format: 'pdf' | 'html' | 'docx';
  template?: string;
  brain_name?: string;
  output_root: string;
}

export interface ExportDoctorResult {
  pandoc_available: boolean;
  pandoc_path?: string;
  xelatex_available: boolean;
  xelatex_path?: string;
  output_dir_default: string;
  template_count: number;
  templates?: ExportTemplate[];
  install_hints?: string[];
}

/**
 * Exercise item from exercises list
 */
export interface ExerciseItemResult {
  id: string;
  name: string;
  context?: string;
  description?: string;
  goal?: string;
  status?: string;
  tags?: string[];
  variant_count: number;
  session_count: number;
  last_session?: string;
  file_path: string;
  rel_path: string;
  created: string;
}

/**
 * Exercise variant
 */
export interface ExerciseVariantResult {
  name?: string;
  description?: string;
  tracking_properties?: Record<string, string>;
}

/**
 * Exercise detail result
 */
export interface ExerciseDetailResult extends ExerciseItemResult {
  variants?: ExerciseVariantResult[];
  duration?: string;
  related?: string[];
  brain_name: string;
  brain_path: string;
}

/**
 * Exercises list result
 */
export interface ExercisesListResult {
  exercises: ExerciseItemResult[];
  brain_name: string;
  brain_path: string;
  total: number;
}

/**
 * Exercise track result
 */
export interface ExerciseTrackResult {
  success: boolean;
  exercise_id: string;
  exercise_name: string;
  journal_path: string;
  date: string;
}

export interface ExerciseMigrateResult {
  success: boolean;
  dry_run: boolean;
  brain_name?: string;
  brain_path?: string;
  files_scanned: number;
  files_changed: number;
  sessions_migrated: number;
  legacy_cli_sessions: number;
  legacy_vscode_sessions: number;
  already_compact: number;
  backups_created: number;
  errors?: string[];
}

/**
 * Exercise new result
 */
export interface ExerciseNewResult {
  success: boolean;
  id: string;
  name: string;
  file_path: string;
  rel_path: string;
}

/**
 * Template info
 */
export interface TemplateInfo {
  brain_type: string;
  template_type: string;
  content: string;
  is_default: boolean;
  path?: string;
}

/**
 * Template list result
 */
export interface TemplateListResult {
  templates: TemplateInfo[];
}

/**
 * Template get result
 */
export interface TemplateGetResult {
  template: TemplateInfo;
}

/**
 * Template reset result
 */
export interface TemplateResetResult {
  message: string;
  template?: TemplateInfo;
}

/**
 * Brain sync result
 */
export interface BrainSyncResult {
  name: string;
  path: string;
  pulled: boolean;
  has_changes: boolean;
  changed_files?: string[];
  committed: boolean;
  commit_message?: string;
  pushed: boolean;
  has_conflicts?: boolean;
  conflict_files?: string[];
  error?: string;
}

/**
 * Sync result from vscode sync command
 */
export interface SyncResult {
  brains: BrainSyncResult[];
}

/**
 * FlipClient wraps the flip CLI for VS Code extension
 */
export class FlipClient {
  private executablePath: string;
  private timeout: number;

  constructor() {
    const config = vscode.workspace.getConfiguration('flip');
    this.executablePath = findFlipExecutable();
    this.timeout = config.get('timeout', 10000);
    console.log(`[Flip] Using executable: ${this.executablePath}`);
  }

  /**
   * Execute a flip command and return parsed JSON result
   */
  private async execute<T>(args: string[], timeoutOverride?: number): Promise<FlipResult<T>> {
    // Add --json flag to args and use execFile instead of shell string concatenation
    const allArgs = [...args, '--json'];
     
    try {
      let stdout = '';
      let stderr = '';
      
      try {
         const result = await execFileAsync(this.executablePath, allArgs, { 
           timeout: timeoutOverride ?? this.timeout,
           env: { ...process.env, TERM_PROGRAM: 'vscode' }
         });
        stdout = result.stdout;
        stderr = result.stderr;
      } catch (execError: any) {
        // Even if command fails, it might have written JSON to stdout
        stdout = execError.stdout || '';
        stderr = execError.stderr || '';
      }

      // Try to parse stdout if it exists (even if command failed)
      if (stdout && stdout.trim() !== '') {
        const parsed = this.parseJsonFromOutput<T>(stdout);
        if (parsed) {
          return parsed;
        }

        // If stdout isn't valid JSON, return it as error
        return {
          success: false,
          command: args[0],
          error: stdout.trim() || stderr?.trim() || 'Invalid JSON response'
        };
      }

      // No stdout - check stderr for error message
      if (stderr && stderr.trim() !== '') {
        return {
          success: false,
          command: args[0],
          error: stderr.trim()
        };
      }

      // Both empty
      return {
        success: false,
        command: args[0],
        error: 'No output from command'
      };

    } catch (error: any) {
      // Check if flip is not installed
      if (error.code === 'ENOENT') {
        const isWindows = process.platform === 'win32';
        const homeDir = os.homedir();
        const goBinPath = path.join(homeDir, 'go', 'bin', isWindows ? 'flip.exe' : 'flip');
        
        return {
          success: false,
          command: args[0],
          error: `flip not found at '${this.executablePath}'. ` +
                 `Install flip with 'go install github.com/httrp/flip@latest' ` +
                 `or set 'flip.executablePath' in VS Code settings to point to your flip binary ` +
                 `(expected location: ${goBinPath})`
        };
      }

      // Timeout
      if (error.killed) {
        return {
          success: false,
          command: args[0],
          error: `Command timed out after ${this.timeout}ms`
        };
      }

      return {
        success: false,
        command: args[0],
        error: error.message || 'Unknown error'
      };
    }
  }

  /**
   * Parses Flip JSON output and tolerates extra log lines around JSON.
   */
  private parseJsonFromOutput<T>(output: string): FlipResult<T> | null {
    const raw = output.trim();
    if (!raw) return null;

    // Fast path: clean JSON
    try {
      return JSON.parse(raw) as FlipResult<T>;
    } catch {
      // Continue with tolerant parsing below
    }

    // Preferred tolerant path: find a JSON object that starts at some '{' and parse to end.
    // This avoids false starts from git output like: rename {a => b}/file.md
    const startIndices: number[] = [];
    for (let i = 0; i < raw.length; i++) {
      if (raw[i] === '{') {
        startIndices.push(i);
      }
    }

    for (const start of startIndices) {
      const candidate = raw.slice(start).trim();
      if (!candidate.startsWith('{')) {
        continue;
      }
      try {
        return JSON.parse(candidate) as FlipResult<T>;
      } catch {
        // Try next possible start
      }
    }

    // Tolerant path: strip any pre/post log lines and parse the JSON object block
    const firstBrace = raw.indexOf('{');
    const lastBrace = raw.lastIndexOf('}');
    if (firstBrace === -1 || lastBrace === -1 || lastBrace <= firstBrace) {
      return null;
    }

    const jsonCandidate = raw.slice(firstBrace, lastBrace + 1);
    try {
      return JSON.parse(jsonCandidate) as FlipResult<T>;
    } catch {
      return null;
    }
  }

  /**
   * Get flip info (version, brains, commands)
   */
  async getInfo(): Promise<FlipResult<FlipInfo>> {
    return this.execute<FlipInfo>(['vscode', 'info']);
  }

  /**
   * Get list of brains in the workspace
   */
  async getBrains(): Promise<FlipResult<BrainInfo[]>> {
    const result = await this.getInfo();
    if (result.success && result.data) {
      return { success: true, data: result.data.brains, command: 'get-brains' };
    }
    return { success: false, error: result.error, command: 'get-brains' };
  }

  /**
   * Get brain name for a given file path by checking which brain contains it
   */
  async getBrainForPath(filePath: string): Promise<string | undefined> {
    const result = await this.getBrains();
    if (!result.success || !result.data) {
      return undefined;
    }

    // Find the brain whose path is a prefix of the file path
    for (const brain of result.data) {
      if (filePath.startsWith(brain.path)) {
        return brain.name;
      }
    }

    return undefined;
  }

  /**
   * Get notes list for a brain
   */
  async getNotes(brain?: string): Promise<FlipResult<NotesResult>> {
    const args = ['vscode', 'notes'];
    if (brain) {
      args.push('--brain', brain);
    }
    return this.execute<NotesResult>(args);
  }

  /**
   * Get status
   */
  async getStatus(): Promise<FlipResult<StatusResult>> {
    return this.execute<StatusResult>(['status']);
  }

  /**
   * Create or open journal for today
   */
  async createJournal(brain?: string, date?: string): Promise<FlipResult<JournalResult>> {
    const args = ['journal', '--no-edit'];
    if (date) {
      args.push('--date', date);
    }
    if (brain) {
      args.push('--brain', brain);
    }
    return this.execute<JournalResult>(args);
  }

  /**
   * Create a new note
   */
  async createNote(options: { title: string; tags?: string; brain?: string }): Promise<FlipResult<NoteResult>> {
    const args = ['note', '--no-edit', '--no-link', '--title', options.title];
    if (options.tags) {
      args.push('--tags', options.tags);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<NoteResult>(args);
  }

  /**
   * Create a quick note
   */
  async createQuicknote(options: { title: string; brain?: string }): Promise<FlipResult<NoteResult>> {
    const args = ['quicknote', '--no-edit', '--no-link', '--title', options.title];
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<NoteResult>(args);
  }

  /**
   * Create a meeting note (non-interactive)
   */
  async createMeeting(options: {
    title: string;
    participants?: string;
    organization?: string;
    project?: string;
    context?: string;
    tags?: string;
    duration?: string;
    series?: string;
    brain?: string;
  }): Promise<FlipResult<NoteResult>> {
    const args = ['meeting-note', '--no-edit', '--no-link', '--title', options.title];
    if (options.participants) args.push('--participants', options.participants);
    if (options.organization) args.push('--organization', options.organization);
    if (options.project) args.push('--project', options.project);
    if (options.context) args.push('--context', options.context);
    if (options.tags) args.push('--tags', options.tags);
    if (options.duration) args.push('--duration', options.duration);
    if (options.series) args.push('--series', options.series);
    if (options.brain) args.push('--brain', options.brain);
    return this.execute<NoteResult>(args);
  }

  /**
   * List meeting series in the active brain
   */
  async listMeetingSeries(): Promise<FlipResult<MeetingSeriesResult>> {
    return this.execute<MeetingSeriesResult>(['vscode', 'meetings', 'list-series']);
  }

  /**
   * List organizations from meeting notes in the active brain (for audit/cleanup)
   */
  async listMeetingOrganizations(): Promise<FlipResult<MeetingOrganizationsResult>> {
    return this.execute<MeetingOrganizationsResult>(['vscode', 'meetings', 'list-organizations']);
  }

  /**
   * List all definitions (organizations, projects, contexts, people)
   */
  async listDefinitions(brain?: string): Promise<FlipResult<DefinitionsResult>> {
    const args = ['vscode', 'definitions', 'list'];
    if (brain) {
      args.push('--brain', brain);
    }
    return this.execute<DefinitionsResult>(args);
  }

  /**
   * Add a new organization to definitions
   */
  async addOrganization(options: { abbreviation: string; name?: string; description?: string; brain?: string }): Promise<FlipResult<DefinitionItem>> {
    const args = ['vscode', 'definitions', 'add-org', '--abbreviation', options.abbreviation];
    if (options.name) {
      args.push('--name', options.name);
    }
    if (options.description) {
      args.push('--description', options.description);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<DefinitionItem>(args);
  }

  /**
   * Add a new person to definitions
   */
  async addPerson(options: { name: string; abbreviation?: string; organization?: string; role?: string; brain?: string }): Promise<FlipResult<DefinitionItem>> {
    const args = ['vscode', 'definitions', 'add-person', '--name', options.name];
    if (options.abbreviation) {
      args.push('--abbreviation', options.abbreviation);
    }
    if (options.organization) {
      args.push('--organization', options.organization);
    }
    if (options.role) {
      args.push('--role', options.role);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<DefinitionItem>(args);
  }

  /**
   * Remove a definition (organization, person, context)
   */
  async removeDefinition(options: { type: string; abbreviation: string; brain?: string }): Promise<FlipResult<void>> {
    const args = ['vscode', 'definitions', 'remove', '--type', options.type, '--abbreviation', options.abbreviation];
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<void>(args);
  }

  /**
   * Get all unique participants from a meeting series history
   */
  async getSeriesParticipants(seriesName: string): Promise<FlipResult<{ participants: string[] }>> {
    return this.execute<{ participants: string[] }>(['vscode', 'meetings', 'get-series-participants', seriesName]);
  }

  /**
   * Check for files missing from the journal
   * @param days Number of days to check (default: 3)
   * @param brain Brain name to check (default: active brain)
   */
  async journalSyncCheck(days?: number, brain?: string): Promise<FlipResult<JournalSyncResult>> {
    const args = ['vscode', 'journal', 'sync-check'];
    if (days !== undefined) {
      args.push('--days', days.toString());
    }
    if (brain) {
      args.push('--brain', brain);
    }
    return this.execute<JournalSyncResult>(args);
  }

  /**
   * Create a new task
   */
  async createTask(options: { 
    description: string; 
    brain?: string; 
    due?: string; 
    priority?: string;
    frog?: boolean;
    file?: string;
    line?: number;
  }): Promise<FlipResult<TaskResult>> {
    const args = ['task', 'new', '--no-edit', '--description', options.description];
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    if (options.due) {
      args.push('--due', options.due);
    }
    if (options.priority) {
      args.push('--priority', options.priority);
    }
    if (options.frog) {
      args.push('--frog');
    }
    if (options.file) {
      args.push('--file', options.file);
    }
    if (options.line !== undefined && options.line > 0) {
      args.push('--line', options.line.toString());
    }
    return this.execute<TaskResult>(args);
  }

  /**
   * Update an existing task status
   */
  async updateTaskStatus(options: {
    file: string;
    line: number;
    status: 'open' | 'in-progress' | 'done' | 'deferred' | 'cancelled';
  }): Promise<FlipResult<TaskResult>> {
    const args = ['task', 'status', '--file', options.file, '--line', options.line.toString(), '--status', options.status];
    return this.execute<TaskResult>(args);
  }

  /**
   * Add a file to the journal
   */
  async addToJournal(options: { file: string; title?: string; type?: string; brain?: string; date?: string }): Promise<FlipResult<any>> {
    const args = ['journal', 'link', '--file', options.file];
    if (options.title) {
      args.push('--title', options.title);
    }
    if (options.type) {
      args.push('--type', options.type);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    if (options.date) {
      args.push('--date', options.date);
    }
    return this.execute<any>(args);
  }

  /**
   * Get tasks with optional filters
   */
  async getTasks(options?: {
    brain?: string;
    status?: string;
    priority?: string;
    frog?: boolean;
    query?: string;
  }): Promise<FlipResult<TasksResult>> {
    const args = ['vscode', 'tasks'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    if (options?.status) {
      args.push('--status', options.status);
    }
    if (options?.priority) {
      args.push('--priority', options.priority);
    }
    if (options?.frog) {
      args.push('--frog');
    }
    if (options?.query) {
      args.push('--query', options.query);
    }
    return this.execute<TasksResult>(args);
  }

  /**
   * Browse tasks with JSON output (for WebView panel)
   */
  async browseTasksJson(): Promise<FlipResult<TasksResult>> {
    const args = ['task', 'browse', '--json'];
    return this.execute<TasksResult>(args);
  }

  /**
   * List tasks with JSON output and optional filters
   */
  async listTasksJson(options?: {
    status?: string;
    priority?: string;
    frog?: boolean;
  }): Promise<FlipResult<TasksResult>> {
    const args = ['task', 'list', '--json'];
    if (options?.status) {
      args.push('--status', options.status);
    }
    if (options?.priority) {
      args.push('--priority', options.priority);
    }
    if (options?.frog) {
      args.push('--frog');
    }
    return this.execute<TasksResult>(args);
  }

  /**
   * Search across all brains
   */
  async search(options: {
    query: string;
    brain?: string;
    type?: 'note' | 'journal' | 'meeting';
    tag?: string;
    limit?: number;
  }): Promise<FlipResult<SearchResults>> {
    const args = ['vscode', 'search', '--query', options.query];
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    if (options.type) {
      args.push('--type', options.type);
    }
    if (options.tag) {
      args.push('--tag', options.tag);
    }
    if (options.limit) {
      args.push('--limit', options.limit.toString());
    }
    return this.execute<SearchResults>(args);
  }

  /**
   * Get recent notes across all brains
   */
  async getRecent(options?: {
    brain?: string;
    type?: 'note' | 'journal' | 'meeting';
    limit?: number;
  }): Promise<FlipResult<SearchResults>> {
    const args = ['vscode', 'recent'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    if (options?.type) {
      args.push('--type', options.type);
    }
    if (options?.limit) {
      args.push('--limit', options.limit.toString());
    }
    return this.execute<SearchResults>(args);
  }

  /**
   * Check brain health (JSON output)
   */
  async checkBrainHealth(options?: {
    brainPath?: string;
    fix?: boolean;
    dryRun?: boolean;
  }): Promise<FlipResult<HealthReport>> {
    const args = ['brain', 'check', 'health'];
    if (options?.brainPath) {
      args.push(options.brainPath);
    }
    if (options?.fix) {
      args.push('--fix');
    }
    if (options?.dryRun) {
      args.push('--dry-run');
    }
    return this.execute<HealthReport>(args);
  }

  /**
   * Normalize media (rename/move assets, update references)
   */
  async normalizeMedia(options?: {
    brainPath?: string;
    fix?: boolean;
    dryRun?: boolean;
  }): Promise<FlipResult<HealthReport>> {
    const args = ['media', 'normalize'];
    if (options?.brainPath) {
      args.push(options.brainPath);
    }
    if (options?.fix) {
      args.push('--fix');
    }
    if (options?.dryRun) {
      args.push('--dry-run');
    }
    return this.execute<HealthReport>(args);
  }

  /**
   * List discovered export templates.
   */
  async listExportTemplates(format: 'pdf' | 'html' | 'docx'): Promise<FlipResult<ExportTemplateListResult>> {
    return this.execute<ExportTemplateListResult>(['export', 'template', 'list', '--format', format]);
  }

  /**
   * Run export dependency checks.
   */
  async exportDoctor(format: 'pdf' | 'html' | 'docx' = 'pdf'): Promise<FlipResult<ExportDoctorResult>> {
    return this.execute<ExportDoctorResult>(['export', 'doctor', '--format', format]);
  }

  /**
   * Convert markdown file via flip export convert.
   */
  async exportConvert(options: {
    input: string;
    format: 'pdf' | 'html' | 'docx';
    template?: string;
    referenceDoc?: string;
    landscape?: boolean;
    openAfter?: boolean;
    output?: string;
    outputDir?: string;
    brain?: string;
  }): Promise<FlipResult<ExportConvertResult>> {
    const args = ['export', 'convert', options.input, '--format', options.format];
    if (options.template) {
      args.push('--template', options.template);
    }
    if (options.referenceDoc) {
      args.push('--reference-doc', options.referenceDoc);
    }
    if (options.landscape) {
      args.push('--landscape');
    }
    if (options.openAfter) {
      args.push('--open');
    }
    if (options.output) {
      args.push('--output', options.output);
    }
    if (options.outputDir) {
      args.push('--output-dir', options.outputDir);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<ExportConvertResult>(args, 120000);
  }

  /**
   * Sync brains bidirectionally: commit local changes, pull remote, push result.
   */
  async sync(options?: {
    brain?: string;
  }): Promise<FlipResult<SyncResult>> {
    const args = ['vscode', 'sync'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<SyncResult>(args);
  }

  /**
   * Get definitions path for a brain
   */
  async getDefinitionsPath(options?: { brain?: string }): Promise<FlipResult<{ path: string }>> {
    const args = ['definitions', 'path'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<{ path: string }>(args);
  }

  /**
   * List all exercises
   */
  async listExercises(options?: { brain?: string; context?: string }): Promise<FlipResult<ExercisesListResult>> {
    const args = ['vscode', 'exercises', 'list'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    if (options?.context) {
      args.push('--context', options.context);
    }
    return this.execute<ExercisesListResult>(args);
  }

  /**
   * Show exercise details
   */
  async showExercise(exerciseId: string, options?: { brain?: string }): Promise<FlipResult<ExerciseDetailResult>> {
    const args = ['vscode', 'exercises', 'show', exerciseId];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<ExerciseDetailResult>(args);
  }

  /**
   * Track an exercise session
   */
  async trackExercise(options: {
    exerciseId: string;
    duration?: number;
    variant?: string;
    notes?: string;
    properties?: Record<string, string | number>;
    brain?: string;
  }): Promise<FlipResult<ExerciseTrackResult>> {
    const args = ['vscode', 'exercises', 'track', '--exercise', options.exerciseId];
    if (options.duration) {
      args.push('--duration', options.duration.toString());
    }
    if (options.variant) {
      args.push('--variant', options.variant);
    }
    if (options.notes) {
      args.push('--notes', options.notes);
    }
    if (options.properties) {
      for (const [key, value] of Object.entries(options.properties)) {
        if (value !== undefined && value !== null && `${value}`.trim() !== '') {
          args.push('--prop', `${key}=${value}`);
        }
      }
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<ExerciseTrackResult>(args);
  }

  /**
   * Migrate legacy exercise journal entries to compact one-line format
   */
  async migrateExerciseSessions(options?: {
    apply?: boolean;
    backup?: boolean;
    days?: number;
    brain?: string;
  }): Promise<FlipResult<ExerciseMigrateResult>> {
    const args = ['vscode', 'exercises', 'migrate-sessions'];
    if (options?.apply) {
      args.push('--apply');
    }
    if (options?.backup === false) {
      args.push('--backup=false');
    }
    if (options?.days && options.days > 0) {
      args.push('--days', options.days.toString());
    }
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<ExerciseMigrateResult>(args);
  }

  /**
   * Create a new exercise
   */
  async createExercise(options: {
    name: string;
    context?: string;
    description?: string;
    goal?: string;
    variants?: ExerciseVariantResult[];
    brain?: string;
  }): Promise<FlipResult<ExerciseNewResult>> {
    const args = ['vscode', 'exercises', 'new', '--name', options.name];
    if (options.context) {
      args.push('--context', options.context);
    }
    if (options.description) {
      args.push('--description', options.description);
    }
    if (options.goal) {
      args.push('--goal', options.goal);
    }
    if (options.variants && options.variants.length > 0) {
      args.push('--variants-json', JSON.stringify(options.variants));
    }
    if (options.brain) {
      args.push('--brain', options.brain);
    }
    return this.execute<ExerciseNewResult>(args);
  }

  /**
   * List templates for a brain type
   */
  async listTemplates(brainType: string): Promise<FlipResult<TemplateListResult>> {
    return this.execute<TemplateListResult>(['vscode', 'templates', 'list', '--brain-type', brainType]);
  }

  /**
   * Get a specific template
   */
  async getTemplate(brainType: string, templateType: string): Promise<FlipResult<TemplateGetResult>> {
    return this.execute<TemplateGetResult>([
      'vscode', 'templates', 'get',
      '--brain-type', brainType,
      '--template-type', templateType
    ]);
  }

  /**
   * Get default template
   */
  async getDefaultTemplate(brainType: string, templateType: string): Promise<FlipResult<TemplateGetResult>> {
    return this.execute<TemplateGetResult>([
      'vscode', 'templates', 'get-default',
      '--brain-type', brainType,
      '--template-type', templateType
    ]);
  }

  /**
   * Reset templates
   */
  async resetTemplates(brainType: string, templateType?: string, all?: boolean): Promise<FlipResult<TemplateResetResult>> {
    const args = ['vscode', 'templates', 'reset', '--brain-type', brainType];
    if (all) {
      args.push('--all');
    } else if (templateType) {
      args.push('--template-type', templateType);
    }
    return this.execute<TemplateResetResult>(args);
  }

  /**
   * Execute arbitrary flip command
   */
  async runCommand(args: string[], timeoutOverride?: number): Promise<FlipResult<any>> {
    return this.execute<any>(args, timeoutOverride);
  }

  /**
   * Check if flip is available
   */
  async isAvailable(): Promise<boolean> {
    try {
      await execFileAsync(this.executablePath, ['--version'], { timeout: 5000 });
      return true;
    } catch {
      return false;
    }
  }
}

// Singleton instance
let _client: FlipClient | undefined;

export function getFlipClient(): FlipClient {
  if (!_client) {
    _client = new FlipClient();
  }
  return _client;
}

/**
 * Refresh client (e.g., after config change)
 */
export function refreshFlipClient(): void {
  _client = new FlipClient();
}
