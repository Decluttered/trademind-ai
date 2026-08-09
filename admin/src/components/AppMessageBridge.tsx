import { App, message as staticMessage } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import { useLayoutEffect } from 'react';

function patchStaticMessage(instance: MessageInstance): () => void {
  const restored: MessageInstance = {
    info: staticMessage.info.bind(staticMessage),
    success: staticMessage.success.bind(staticMessage),
    error: staticMessage.error.bind(staticMessage),
    warning: staticMessage.warning.bind(staticMessage),
    loading: staticMessage.loading.bind(staticMessage),
    open: staticMessage.open.bind(staticMessage),
    destroy: staticMessage.destroy.bind(staticMessage),
  };

  staticMessage.info = (...args) => instance.info(...args);
  staticMessage.success = (...args) => instance.success(...args);
  staticMessage.error = (...args) => instance.error(...args);
  staticMessage.warning = (...args) => instance.warning(...args);
  staticMessage.loading = (...args) => instance.loading(...args);
  staticMessage.open = (...args) => instance.open(...args);
  staticMessage.destroy = (...args) => instance.destroy(...args);

  return () => {
    staticMessage.info = restored.info;
    staticMessage.success = restored.success;
    staticMessage.error = restored.error;
    staticMessage.warning = restored.warning;
    staticMessage.loading = restored.loading;
    staticMessage.open = restored.open;
    staticMessage.destroy = restored.destroy;
  };
}

/** Patches antd static `message.*` to use App context (theme + cssVar). Mount inside `<App>`. */
export default function AppMessageBridge() {
  const { message } = App.useApp();
  useLayoutEffect(() => patchStaticMessage(message), [message]);
  return null;
}
