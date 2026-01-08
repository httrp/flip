/**
 * Search commands for flip extension
 * 
 * Key differentiator: Searches across ALL brains, not just open workspace
 * This is what VS Code's built-in search cannot do!
 */
import * as vscode from 'vscode';
import { getFlipClient, SearchItem } from '../flip-client';

/**
 * Convert a SearchItem to a QuickPickItem
 */
function searchItemToQuickPick(item: SearchItem): vscode.QuickPickItem & { item: SearchItem } {
  const typeIcon = {
    note: '📝',
    journal: '📅',
    meeting: '👥',
    task: '✅'
  }[item.type] || '📄';

  const brainLabel = `[${item.brain_name}]`;
  
  let description = `${typeIcon} ${brainLabel} ${item.rel_path}`;
  if (item.tags && item.tags.length > 0) {
    description += ` • ${item.tags.map(t => `#${t}`).join(' ')}`;
  }

  let detail: string | undefined;
  if (item.match_text) {
    detail = `📍 L${item.match_line}: ${item.match_text}`;
  }

  return {
    label: item.title,
    description,
    detail,
    item
  };
}

/**
 * Search across all brains
 * 
 * Unlike VS Code's built-in search, this searches ALL configured brains,
 * even those not open in the current workspace!
 */
export async function searchAllBrains(): Promise<void> {
  const client = getFlipClient();

  // Create a QuickPick for interactive search
  const quickPick = vscode.window.createQuickPick<vscode.QuickPickItem & { item?: SearchItem }>();
  quickPick.title = '🔍 Search All Brains';
  quickPick.placeholder = 'Type to search across all brains...';
  quickPick.matchOnDescription = true;
  quickPick.matchOnDetail = true;

  let searchTimeout: NodeJS.Timeout | undefined;

  // Handle input changes with debounce
  quickPick.onDidChangeValue(async (value) => {
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }

    if (!value || value.length < 2) {
      quickPick.items = [];
      return;
    }

    searchTimeout = setTimeout(async () => {
      quickPick.busy = true;

      const result = await client.search({ query: value, limit: 50 });

      if (result.success && result.data) {
        quickPick.items = result.data.results.map(searchItemToQuickPick);
      } else {
        quickPick.items = [{
          label: '$(warning) Search failed',
          description: result.error || 'Unknown error'
        }];
      }

      quickPick.busy = false;
    }, 300); // Debounce 300ms
  });

  // Handle selection
  quickPick.onDidAccept(async () => {
    const selected = quickPick.selectedItems[0];
    if (selected && 'item' in selected && selected.item) {
      quickPick.hide();
      await openSearchResult(selected.item);
    }
  });

  quickPick.onDidHide(() => quickPick.dispose());
  quickPick.show();
}

/**
 * Show recent notes across all brains
 * 
 * Like Cmd+E in JetBrains IDEs - quick access to recently modified files
 */
export async function showRecentNotes(): Promise<void> {
  const client = getFlipClient();

  const quickPick = vscode.window.createQuickPick<vscode.QuickPickItem & { item?: SearchItem }>();
  quickPick.title = '📋 Recent Notes';
  quickPick.placeholder = 'Select a recent note to open...';
  quickPick.busy = true;
  quickPick.show();

  // Fetch recent notes
  const result = await client.getRecent({ limit: 30 });

  if (result.success && result.data) {
    quickPick.items = result.data.results.map(item => {
      const pick = searchItemToQuickPick(item);
      // Add modification time to description
      const modDate = new Date(item.modified);
      const timeAgo = formatTimeAgo(modDate);
      pick.description = `${pick.description} • ${timeAgo}`;
      return pick;
    });
  } else {
    quickPick.items = [{
      label: '$(warning) Failed to get recent notes',
      description: result.error || 'Unknown error'
    }];
  }

  quickPick.busy = false;

  // Handle selection
  quickPick.onDidAccept(async () => {
    const selected = quickPick.selectedItems[0];
    if (selected && 'item' in selected && selected.item) {
      quickPick.hide();
      await openSearchResult(selected.item);
    }
  });

  quickPick.onDidHide(() => quickPick.dispose());
}

/**
 * Search with filters (type, brain, tag)
 */
export async function searchWithFilters(): Promise<void> {
  const client = getFlipClient();

  // First, let user select filters
  const filterOptions: vscode.QuickPickItem[] = [
    { label: '$(search) Search all', description: 'Search across all brains with no filters' },
    { label: '$(file) Filter by type', description: 'Search only notes, journals, or meetings' },
    { label: '$(database) Filter by brain', description: 'Search within a specific brain' },
    { label: '$(tag) Filter by tag', description: 'Search for specific tags' }
  ];

  const filterChoice = await vscode.window.showQuickPick(filterOptions, {
    title: '🔍 Search Options',
    placeHolder: 'How would you like to search?'
  });

  if (!filterChoice) {
    return;
  }

  let searchOptions: { brain?: string; type?: 'note' | 'journal' | 'meeting'; tag?: string } = {};

  if (filterChoice.label.includes('Filter by type')) {
    const typeChoice = await vscode.window.showQuickPick([
      { label: '📝 Notes', value: 'note' },
      { label: '📅 Journals', value: 'journal' },
      { label: '👥 Meetings', value: 'meeting' }
    ], {
      title: 'Select type',
      placeHolder: 'Which type of notes?'
    });
    if (typeChoice) {
      searchOptions.type = (typeChoice as any).value;
    }
  } else if (filterChoice.label.includes('Filter by brain')) {
    const infoResult = await client.getInfo();
    if (infoResult.success && infoResult.data) {
      const brainChoice = await vscode.window.showQuickPick(
        infoResult.data.brains.map(b => ({
          label: b.name,
          description: b.path
        })),
        {
          title: 'Select brain',
          placeHolder: 'Which brain to search?'
        }
      );
      if (brainChoice) {
        searchOptions.brain = brainChoice.label;
      }
    }
  } else if (filterChoice.label.includes('Filter by tag')) {
    const tagInput = await vscode.window.showInputBox({
      title: 'Enter tag',
      placeHolder: 'e.g., work, project, meeting'
    });
    if (tagInput) {
      searchOptions.tag = tagInput;
    }
  }

  // Now ask for search query
  const query = await vscode.window.showInputBox({
    title: '🔍 Search Query',
    placeHolder: 'Enter search terms...'
  });

  if (!query) {
    return;
  }

  // Execute search
  const result = await client.search({ query, ...searchOptions, limit: 50 });

  if (!result.success || !result.data) {
    vscode.window.showErrorMessage(`Search failed: ${result.error || 'Unknown error'}`);
    return;
  }

  // Show results
  const items = result.data.results.map(searchItemToQuickPick);

  const selected = await vscode.window.showQuickPick(items, {
    title: `🔍 Search Results (${result.data.total_count} found)`,
    placeHolder: 'Select a note to open...',
    matchOnDescription: true,
    matchOnDetail: true
  });

  if (selected && selected.item) {
    await openSearchResult(selected.item);
  }
}

/**
 * Open a search result in the editor
 */
async function openSearchResult(item: SearchItem): Promise<void> {
  const uri = vscode.Uri.file(item.path);
  
  try {
    const document = await vscode.workspace.openTextDocument(uri);
    const editor = await vscode.window.showTextDocument(document);

    // If there's a match line, go to it
    if (item.match_line && item.match_line > 0) {
      const line = Math.max(0, item.match_line - 1);
      const position = new vscode.Position(line, 0);
      editor.selection = new vscode.Selection(position, position);
      editor.revealRange(
        new vscode.Range(position, position),
        vscode.TextEditorRevealType.InCenter
      );
    }
  } catch (error: any) {
    vscode.window.showErrorMessage(`Failed to open file: ${error.message}`);
  }
}

/**
 * Format a date as "time ago"
 */
function formatTimeAgo(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) {
    return 'just now';
  } else if (diffMins < 60) {
    return `${diffMins}m ago`;
  } else if (diffHours < 24) {
    return `${diffHours}h ago`;
  } else if (diffDays === 1) {
    return 'yesterday';
  } else if (diffDays < 7) {
    return `${diffDays}d ago`;
  } else {
    return date.toLocaleDateString();
  }
}
