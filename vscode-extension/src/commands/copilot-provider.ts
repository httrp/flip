/**
 * copilot-provider.ts — GitHub Copilot integration via VS Code Language Model API
 *
 * Uses vscode.lm.selectChatModels() to discover available models and
 * vscode.LanguageModelChat.sendRequest() to run prompts through Copilot.
 *
 * This allows flip to use whatever models the user's Copilot subscription
 * provides (GPT-4o, Claude, etc.) without needing a separate API key.
 */

import * as vscode from 'vscode';

/** Minimal model descriptor returned by getCopilotModels(). */
export interface CopilotModel {
  /** Stable identifier passed to selectChatModels({id}) */
  id: string;
  /** Human-readable label for the picker */
  label: string;
  /** Vendor (e.g. "copilot") */
  vendor: string;
  /** Model family (e.g. "gpt-4o", "claude-3.5-sonnet") */
  family: string;
  /** Max input tokens the model supports */
  maxInputTokens: number;
}

/**
 * Check whether the VS Code Language Model API is available
 * and at least one chat model can be selected.
 */
export async function isCopilotAvailable(): Promise<boolean> {
  try {
    if (typeof vscode.lm === 'undefined' || typeof vscode.lm.selectChatModels !== 'function') {
      return false;
    }
    const models = await vscode.lm.selectChatModels();
    return models.length > 0;
  } catch {
    return false;
  }
}

/**
 * Return the list of language models currently available through Copilot.
 * Returns an empty array when the API is unavailable or no models exist.
 */
export async function getCopilotModels(): Promise<CopilotModel[]> {
  try {
    if (typeof vscode.lm === 'undefined') {
      return [];
    }
    const models = await vscode.lm.selectChatModels();
    return models.map((m) => ({
      id: m.id,
      label: `${m.name} (${m.vendor})`,
      vendor: m.vendor,
      family: m.family,
      maxInputTokens: m.maxInputTokens,
    }));
  } catch {
    return [];
  }
}

/**
 * Send a completion request to a Copilot model and collect the full response.
 *
 * @param systemPrompt - System-level instruction
 * @param userPrompt   - User message / content
 * @param modelId      - The model id from CopilotModel.id
 * @param token        - Cancellation token (optional)
 * @returns The full response text
 */
export async function sendCopilotRequest(
  systemPrompt: string,
  userPrompt: string,
  modelId: string,
  token?: vscode.CancellationToken,
): Promise<string> {
  // Select the specific model
  const models = await vscode.lm.selectChatModels({ id: modelId });
  if (models.length === 0) {
    throw new Error(`Copilot model not found: ${modelId}`);
  }
  const model = models[0];

  // Build the message array
  const messages = [
    vscode.LanguageModelChatMessage.User(systemPrompt),
    vscode.LanguageModelChatMessage.User(userPrompt),
  ];

  // Send the request
  const response = await model.sendRequest(
    messages,
    {},
    token ?? new vscode.CancellationTokenSource().token,
  );

  // Collect the streamed response
  let result = '';
  for await (const chunk of response.text) {
    result += chunk;
  }
  return result;
}

/**
 * Send a completion request with real-time progress reporting.
 *
 * @param systemPrompt  - System-level instruction
 * @param userPrompt    - User message / content
 * @param modelId       - The model id from CopilotModel.id
 * @param progress      - VS Code progress reporter
 * @param token         - Cancellation token
 * @returns The full response text
 */
export async function sendCopilotRequestWithProgress(
  systemPrompt: string,
  userPrompt: string,
  modelId: string,
  progress: vscode.Progress<{ message?: string; increment?: number }>,
  token: vscode.CancellationToken,
): Promise<string> {
  const models = await vscode.lm.selectChatModels({ id: modelId });
  if (models.length === 0) {
    throw new Error(`Copilot model not found: ${modelId}`);
  }
  const model = models[0];

  const messages = [
    vscode.LanguageModelChatMessage.User(systemPrompt),
    vscode.LanguageModelChatMessage.User(userPrompt),
  ];

  progress.report({ message: `Sending to ${model.name}…` });

  const response = await model.sendRequest(messages, {}, token);

  let result = '';
  let chunks = 0;
  for await (const chunk of response.text) {
    result += chunk;
    chunks++;
    if (chunks % 10 === 0) {
      progress.report({ message: `Generating… (${result.length} chars)` });
    }
  }

  return result;
}
