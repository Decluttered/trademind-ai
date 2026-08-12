import { App, Modal as staticModal } from 'antd';
import type { HookAPI as ModalHookApi } from 'antd/es/modal/useModal';
import { useLayoutEffect } from 'react';

function patchStaticModal(instance: ModalHookApi): () => void {
  const restored = {
    info: staticModal.info.bind(staticModal),
    success: staticModal.success.bind(staticModal),
    error: staticModal.error.bind(staticModal),
    warning: staticModal.warning.bind(staticModal),
    confirm: staticModal.confirm.bind(staticModal),
  };

  staticModal.info = (...args) => instance.info(...args);
  staticModal.success = (...args) => instance.success(...args);
  staticModal.error = (...args) => instance.error(...args);
  staticModal.warning = (...args) => instance.warning(...args);
  staticModal.confirm = (...args) => instance.confirm(...args);

  return () => {
    staticModal.info = restored.info;
    staticModal.success = restored.success;
    staticModal.error = restored.error;
    staticModal.warning = restored.warning;
    staticModal.confirm = restored.confirm;
  };
}

/** Routes existing static `Modal.*` calls through the themed Ant Design App context. */
export default function AppModalBridge() {
  const { modal } = App.useApp();
  useLayoutEffect(() => patchStaticModal(modal), [modal]);
  return null;
}
