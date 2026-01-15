import { exec } from 'child_process';
import { promisify } from 'util';
import * as vscode from 'vscode';

const execAsync = promisify(exec);

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

/**
 * Brain sync result
 */
export interface BrainSyncResult {
  name: string;
  path: string;
  has_changes: boolean;
  changed_files?: string[];
  committed: boolean;
  commit_message?: string;
  pushed: boolean;
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
    this.executablePath = config.get('executablePath', 'flip');
    this.timeout = config.get('timeout', 10000);
  }

  /**
   * Escape a string for safe shell usage
   */
  private shellEscape(str: string): string {
    // Use single quotes and escape any single quotes within
    return `'${str.replace(/'/g, "'\\''")}'`;
  }

  /**
   * Execute a flip command and return parsed JSON result
   */
  private async execute<T>(args: string[]): Promise<FlipResult<T>> {
    const cmd = `${this.executablePath} ${args.join(' ')} --json`;
    
    try {
      let stdout = '';
      let stderr = '';
      
      try {
        const result = await execAsync(cmd, { 
          timeout: this.timeout,
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
        try {
          const result = JSON.parse(stdout);
          return result as FlipResult<T>;
        } catch (parseError) {
          // If stdout isn't valid JSON, return it as error
          return {
            success: false,
            command: args[0],
            error: stdout.trim() || stderr?.trim() || 'Invalid JSON response'
          };
        }
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
        return {
          success: false,
          command: args[0],
          error: 'flip not found. Please install flip and ensure it\'s in your PATH.'
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
    const args = ['note', '--no-edit', '--no-link', '--title', this.shellEscape(options.title)];
    if (options.tags) {
      args.push('--tags', this.shellEscape(options.tags));
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
    const args = ['quicknote', '--no-edit', '--no-link', '--title', this.shellEscape(options.title)];
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
    const args = ['meeting-note', '--no-edit', '--no-link', '--title', this.shellEscape(options.title)];
    if (options.participants) args.push('--participants', this.shellEscape(options.participants));
    if (options.organization) args.push('--organization', this.shellEscape(options.organization));
    if (options.project) args.push('--project', this.shellEscape(options.project));
    if (options.context) args.push('--context', this.shellEscape(options.context));
    if (options.tags) args.push('--tags', this.shellEscape(options.tags));
    if (options.duration) args.push('--duration', this.shellEscape(options.duration));
    if (options.series) args.push('--series', this.shellEscape(options.series));
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
      args.push('--brain', this.shellEscape(brain));
    }
    return this.execute<DefinitionsResult>(args);
  }

  /**
   * Add a new organization to definitions
   */
  async addOrganization(options: { abbreviation: string; name?: string; description?: string; brain?: string }): Promise<FlipResult<DefinitionItem>> {
    const args = ['vscode', 'definitions', 'add-org', '--abbreviation', this.shellEscape(options.abbreviation)];
    if (options.name) {
      args.push('--name', this.shellEscape(options.name));
    }
    if (options.description) {
      args.push('--description', this.shellEscape(options.description));
    }
    if (options.brain) {
      args.push('--brain', this.shellEscape(options.brain));
    }
    return this.execute<DefinitionItem>(args);
  }

  /**
   * Add a new person to definitions
   */
  async addPerson(options: { name: string; abbreviation?: string; organization?: string; role?: string; brain?: string }): Promise<FlipResult<DefinitionItem>> {
    const args = ['vscode', 'definitions', 'add-person', '--name', this.shellEscape(options.name)];
    if (options.abbreviation) {
      args.push('--abbreviation', this.shellEscape(options.abbreviation));
    }
    if (options.organization) {
      args.push('--organization', this.shellEscape(options.organization));
    }
    if (options.role) {
      args.push('--role', this.shellEscape(options.role));
    }
    if (options.brain) {
      args.push('--brain', this.shellEscape(options.brain));
    }
    return this.execute<DefinitionItem>(args);
  }

  /**
   * Remove a definition (organization, person, context)
   */
  async removeDefinition(options: { type: string; abbreviation: string; brain?: string }): Promise<FlipResult<void>> {
    const args = ['vscode', 'definitions', 'remove', '--type', options.type, '--abbreviation', this.shellEscape(options.abbreviation)];
    if (options.brain) {
      args.push('--brain', this.shellEscape(options.brain));
    }
    return this.execute<void>(args);
  }

  /**
   * Get all unique participants from a meeting series history
   */
  async getSeriesParticipants(seriesName: string): Promise<FlipResult<{ participants: string[] }>> {
    return this.execute<{ participants: string[] }>(['vscode', 'meetings', 'get-series-participants', this.shellEscape(seriesName)]);
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
    const args = ['task', 'new', '--no-edit', '--description', this.shellEscape(options.description)];
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
    const args = ['journal', 'link', '--file', this.shellEscape(options.file)];
    if (options.title) {
      args.push('--title', this.shellEscape(options.title));
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
   * Search across all brains
   */
  async search(options: {
    query: string;
    brain?: string;
    type?: 'note' | 'journal' | 'meeting';
    tag?: string;
    limit?: number;
  }): Promise<FlipResult<SearchResults>> {
    const args = ['vscode', 'search', '--query', this.shellEscape(options.query)];
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
      args.push(this.shellEscape(options.brainPath));
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
      args.push(this.shellEscape(options.brainPath));
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
   * Sync (commit and optionally push) changes in brains
   */
  async sync(options?: {
    brain?: string;
    push?: boolean;
  }): Promise<FlipResult<SyncResult>> {
    const args = ['vscode', 'sync'];
    if (options?.brain) {
      args.push('--brain', options.brain);
    }
    if (options?.push) {
      args.push('--push');
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
   * Check if flip is available
   */
  async isAvailable(): Promise<boolean> {
    try {
      await execAsync(`${this.executablePath} --version`, { timeout: 5000 });
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
