import {
  PLATFORM_DISPLAY_LABEL,
  SUPPORTED_COMMERCE_PLATFORM_KEYS,
} from '@/constants/platformLabels';

/** AI skill template - use-case label mapping (matches backend ai_prompts.scene) */
export const AI_PROMPT_SCENE_LABEL: Record<string, string> = {
  product: '商品优化',
  customer_service: '智能客服',
  collect: '商品采集',
};

export function aiPromptSceneLabel(scene?: string): string {
  const k = (scene || '').trim();
  if (!k) return '—';
  return AI_PROMPT_SCENE_LABEL[k] || k;
}

export const AI_PROMPT_SCENE_OPTIONS = Object.entries(AI_PROMPT_SCENE_LABEL).map(([value, label]) => ({
  value,
  label,
}));

/** AI text provider label mapping (matches the provider under "Settings -> AI") */
export const AI_TEXT_PROVIDER_LABEL: Record<string, string> = {
  openai: 'OpenAI',
  openai_compatible: 'OpenAI Compatible',
  deepseek: 'DeepSeek',
  qwen: '通义千问',
  doubao: '豆包',
  gemini: 'Gemini',
  claude: 'Claude',
  ollama: 'Ollama',
};

export function aiTextProviderLabel(provider?: string): string {
  const k = (provider || '').trim().toLowerCase().replace(/-/g, '_');
  if (!k) return '';
  return AI_TEXT_PROVIDER_LABEL[k] || provider || '';
}

export const AI_TEXT_PROVIDER_OPTIONS = Object.entries(AI_TEXT_PROVIDER_LABEL).map(([value, label]) => ({
  value,
  label,
}));

export const AI_PROMPT_USE_SYSTEM_DEFAULT = '跟随系统默认';

/** AI generation language: label shown to the user, value stays as the standard language code used by the Prompt. */
export const AI_LANGUAGE_OPTIONS = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: '英语' },
  { value: 'th', label: '泰语' },
  { value: 'vi', label: '越南语' },
  { value: 'id', label: '印度尼西亚语' },
  { value: 'ms', label: '马来语' },
  { value: 'ja', label: '日语' },
  { value: 'ko', label: '韩语' },
  { value: 'de', label: '德语' },
  { value: 'fr', label: '法语' },
  { value: 'es', label: '西班牙语' },
  { value: 'pt', label: '葡萄牙语' },
];

/** AI copy tone: label shown to the user, value stays compatible with existing Prompt variables. */
export const AI_TONE_OPTIONS = [
  { value: 'professional', label: '专业稳健' },
  { value: 'friendly', label: '亲切自然' },
  { value: 'concise', label: '简洁直接' },
  { value: 'persuasive', label: '卖点导向' },
  { value: 'enthusiastic', label: '热情活力' },
  { value: 'formal', label: '正式严谨' },
];

/** AI product copy target platform; the display name doubles as the Prompt value, for compatibility with the existing request payload. */
export const AI_TARGET_PLATFORM_OPTIONS = [
  { value: '跨境通用', label: '跨境通用' },
  ...SUPPORTED_COMMERCE_PLATFORM_KEYS.map((platform) => ({
    value: PLATFORM_DISPLAY_LABEL[platform],
    label: PLATFORM_DISPLAY_LABEL[platform],
  })),
];

/** AI conversation context platform uses the internal platform code, and keeps the manual conversation option. */
export const AI_CONTEXT_PLATFORM_OPTIONS = [
  ...SUPPORTED_COMMERCE_PLATFORM_KEYS,
  'manual',
].map((platform) => ({
  value: platform,
  label: PLATFORM_DISPLAY_LABEL[platform],
}));

/** AI task record - task type label mapping (matches backend ai_tasks.task_type) */
export const AI_TASK_TYPE_LABEL: Record<string, string> = {
  title_optimize: '标题优化',
  product_description_generate: '商品描述生成',
  customer_reply_generate: '客服回复建议',
  collect_rule_generate: '采集规则生成',
};

/** AI task record - skill template code label mapping (matches backend ai_prompts.code) */
export const AI_PROMPT_CODE_LABEL: Record<string, string> = {
  product_title_optimize: '商品标题优化',
  product_description_generate: '商品描述生成',
  customer_reply_generate: 'AI 客服回复建议',
  collect_rule_generate: 'AI 生成自定义采集规则',
};

export function aiTaskTypeLabel(taskType?: string): string {
  const k = (taskType || '').trim();
  if (!k) return '—';
  return AI_TASK_TYPE_LABEL[k] || k;
}

export function aiPromptCodeLabel(code?: string): string {
  const k = (code || '').trim();
  if (!k) return '—';
  return AI_PROMPT_CODE_LABEL[k] || k;
}

export const AI_TASK_TYPE_OPTIONS = Object.entries(AI_TASK_TYPE_LABEL).map(([value, label]) => ({
  value,
  label,
}));

export const AI_PROMPT_CODE_OPTIONS = Object.entries(AI_PROMPT_CODE_LABEL).map(([value, label]) => ({
  value,
  label,
}));
