import { useModel } from '@umijs/max';
import type { InitialStateModel } from '@/typings/umi-runtime';

/** Type-safe Umi `@@initialState` model (avoids useModel returning unknown by default). */
export function useInitialStateModel(): InitialStateModel {
  return useModel('@@initialState') as InitialStateModel;
}
