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
  brain_name: string;
  brain_path: string;
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
   * Get status
   */
  async getStatus(): Promise<FlipResult<StatusResult>> {
    return this.execute<StatusResult>(['status']);
  }

  /**
   * Create or open journal for today
   */
  async createJournal(options?: { date?: string; brain?: string }): Promise<FlipResult<JournalResult>> {
    const args = ['journal', '--no-edit'];
    if (options?.date) {
      args.push('--date', options.date);
    }
    if (options?.brain) {
      args.push('--brain', options.brain);
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
    const args = ['quicknote', '--no-edit', '--title', `"${options.title}"`];
    if (options.brain) {
      args.push('--brain', options.brain);
    }
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
    return this.execute<TaskResult>(args);
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
