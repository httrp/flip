import * as vscode from 'vscode';
import { getFlipClient, BrainInfo } from '../flip-client';

const output = vscode.window.createOutputChannel('Flip Relocate');

export async function brainRelocate(): Promise<void> {
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

	// Step 1: Select source brain
	interface BrainPickItem extends vscode.QuickPickItem {
		brain: BrainInfo;
	}

	const brainItems: BrainPickItem[] = brains.map((b) => ({
		label: `$(database) ${b.name}`,
		description: b.active ? '(aktiv)' : '',
		detail: b.path,
		brain: b,
	}));

	const selectedBrain = await vscode.window.showQuickPick(brainItems, {
		placeHolder: 'Select brain to relocate',
	});

	if (!selectedBrain) {
		return;
	}

	// Step 2: Get destination path
	const destPath = await vscode.window.showInputBox({
		prompt: 'Enter new destination path for the brain',
		value: selectedBrain.brain.path.replace(selectedBrain.brain.name, ''),
		placeHolder: '~/new-location/my-brain',
		validateInput: (value: string) => {
			if (!value.trim()) {
				return 'Destination path cannot be empty';
			}
			return '';
		},
	});

	if (!destPath) {
		return;
	}

	// Step 3: Choose mode (normal or dry-run)
	interface ModePickItem extends vscode.QuickPickItem {
		mode: 'execute' | 'dry-run';
	}

	const modeItems: ModePickItem[] = [
		{
			label: '$(play) Execute Relocation',
			description: 'Move brain to new location and verify',
			mode: 'execute',
		},
		{
			label: '$(eye) Dry Run Preview',
			description: 'Show what would happen without making changes',
			mode: 'dry-run',
		},
	];

	const selectedMode = await vscode.window.showQuickPick(modeItems, {
		placeHolder: 'Choose relocation mode',
	});

	if (!selectedMode) {
		return;
	}

	// Step 4: Confirm action
	const confirmMsg =
		selectedMode.mode === 'dry-run'
			? `Preview relocating "${selectedBrain.brain.name}" to "${destPath}"?`
			: `Relocate "${selectedBrain.brain.name}" to "${destPath}"?`;

	const confirmed = await vscode.window.showWarningMessage(
		confirmMsg,
		{ modal: true },
		'Yes',
		'Cancel'
	);

	if (confirmed !== 'Yes') {
		return;
	}

	// Step 5: Run relocation
	const progressTitle =
		selectedMode.mode === 'dry-run'
			? 'Previewing brain relocation...'
			: 'Relocating brain...';

	const cmdArgs: string[] = ['brain', 'relocate', selectedBrain.brain.path, destPath, '--force'];
	if (selectedMode.mode === 'dry-run') {
		cmdArgs.push('--dry-run');
	}

	const result = await vscode.window.withProgress(
		{
			location: vscode.ProgressLocation.Notification,
			title: progressTitle,
			cancellable: false,
		},
		async () => {
			return await client.runCommand(cmdArgs);
		}
	);

	if (result.success) {
		const msg =
			selectedMode.mode === 'dry-run'
				? 'Dry run preview completed'
				: 'Brain relocated successfully';
		vscode.window.showInformationMessage(msg);
		output.show(true);
	} else {
		const errorMsg = result.error ? (typeof result.error === 'string' ? result.error : JSON.stringify(result.error)) : 'Unknown error';
		vscode.window.showErrorMessage(`Relocation failed: ${errorMsg}`);
		output.show(true);
	}
}
