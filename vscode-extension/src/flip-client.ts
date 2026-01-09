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
 * Task creation result
 */
export interface TaskResult {
  action: string;
  path: string;
  description: string;
  due?: string;
  priority?: string;
  frog?: boolean;
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
   * Execute a flip command and return parsed JSON result
   */
  private async execute<T>(args: string[]): Promise<FlipResult<T>> {
    const cmd = `${this.executablePath} ${args.join(' ')} --json`;
    
    try {
      const { stdout, stderr } = await execAsync(cmd, { 
        timeout: this.timeout,
        env: { ...process.env, TERM_PROGRAM: 'vscode' }
      });

      if (stderr && !stdout) {
        return {
          success: false,
          command: args[0],
          error: stderr.trim()
        };
      }

      const result = JSON.parse(stdout);
      return result as FlipResult<T>;

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
    const args = ['note', '--no-edit', '--no-link', '--title', `"${options.title}"`];
    if (options.tags) {
      args.push('--tags', `"${options.tags}"`);
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
    const args = ['quicknote', '--no-edit', '--no-link', '--title', `"${options.title}"`];
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
    const args = ['meeting-note', '--no-edit', '--no-link', '--title', `"${options.title}"`];
    if (options.participants) args.push('--participants', `"${options.participants}"`);
    if (options.organization) args.push('--organization', `"${options.organization}"`);
    if (options.project) args.push('--project', `"${options.project}"`);
    if (options.context) args.push('--context', options.context);
    if (options.tags) args.push('--tags', `"${options.tags}"`);
    if (options.duration) args.push('--duration', `"${options.duration}"`);
    if (options.series) args.push('--series', `"${options.series}"`);
    if (options.brain) args.push('--brain', options.brain);
    return this.execute<NoteResult>(args);
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
    const args = ['task', 'new', '--no-edit', '--no-link', '--description', `"${options.description}"`];
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
   * Add a file to today's journal
   */
  async addToJournal(options: { file: string; title?: string; type?: string; brain?: string }): Promise<FlipResult<any>> {
    const args = ['journal', 'link', '--file', `"${options.file}"`];
    if (options.title) {
      args.push('--title', `"${options.title}"`);
    }
    if (options.type) {
      args.push('--type', options.type);
    }
    if (options.brain) {
      args.push('--brain', options.brain);
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
    const args = ['vscode', 'search', '--query', `"${options.query}"`];
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
