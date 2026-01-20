import * as vscode from 'vscode';

export async function pickDate(defaultDate: string, title: string = 'Select Date'): Promise<string | undefined> {
  return new Promise((resolve) => {
    const panel = vscode.window.createWebviewPanel(
      'flipDatePicker',
      title,
      vscode.ViewColumn.Active,
      { enableScripts: true, retainContextWhenHidden: false }
    );

    const nonce = getNonce();
    const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src https: data:; style-src 'unsafe-inline'; script-src 'nonce-${nonce}';" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, Oxygen, Ubuntu, Cantarell, sans-serif; margin: 16px; }
    .row { display: flex; gap: 8px; align-items: center; }
    input[type=date] { padding: 6px 8px; font-size: 14px; }
    button { padding: 6px 12px; font-size: 14px; }
    .spacer { flex: 1; }
  </style>
  </head>
  <body>
    <div class="row">
      <label for="date">Date:</label>
      <input id="date" type="date" value="${defaultDate}" />
      <span class="spacer"></span>
      <button id="cancel">Cancel</button>
      <button id="ok">OK</button>
    </div>
    <script nonce="${nonce}">
      const vscode = acquireVsCodeApi();
      const dateEl = document.getElementById('date');
      const okBtn = document.getElementById('ok');
      const cancelBtn = document.getElementById('cancel');
      okBtn.addEventListener('click', () => {
        const v = dateEl.value;
        vscode.postMessage({ type: 'ok', value: v });
      });
      cancelBtn.addEventListener('click', () => {
        vscode.postMessage({ type: 'cancel' });
      });
      window.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') okBtn.click();
        if (e.key === 'Escape') cancelBtn.click();
      });
    </script>
  </body>
  </html>`;

    panel.webview.html = html;
    const sub = panel.webview.onDidReceiveMessage((msg: any) => {
      if (msg?.type === 'ok' && typeof msg.value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(msg.value)) {
        resolve(msg.value);
      } else {
        resolve(undefined);
      }
      sub.dispose();
      panel.dispose();
    });
    panel.onDidDispose(() => {
      resolve(undefined);
      sub.dispose();
    });
  });
}

function getNonce(): string {
  let text = '';
  const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  for (let i = 0; i < 32; i++) {
    text += possible.charAt(Math.floor(Math.random() * possible.length));
  }
  return text;
}
