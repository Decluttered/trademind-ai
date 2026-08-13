export type ActionGuard = {
  tryLock: (key: string) => boolean;
  unlock: (key: string) => void;
  isLocked: (key: string) => boolean;
};

export function createActionGuard(): ActionGuard {
  const locks = new Set<string>();
  return {
    tryLock(key) {
      if (locks.has(key)) return false;
      locks.add(key);
      return true;
    },
    unlock(key) {
      locks.delete(key);
    },
    isLocked(key) {
      return locks.has(key);
    },
  };
}
