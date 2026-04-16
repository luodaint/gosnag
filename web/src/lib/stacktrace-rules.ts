import type { StacktraceRules } from '@/lib/api'

export type StackFrameKind = 'app' | 'framework' | 'external'

export type StacktracePresetDefinition = {
  id: string
  label: string
  description: string
  rules: StacktraceRules
}

export const STACKTRACE_RULE_PRESETS: StacktracePresetDefinition[] = [
  {
    id: 'generic',
    label: 'Generic',
    description: 'Minimal defaults. Useful when you want to define everything manually.',
    rules: { preset: 'generic', app_patterns: [], framework_patterns: [], external_patterns: [] },
  },
  {
    id: 'codeigniter',
    label: 'CodeIgniter',
    description: 'Highlights application code over CodeIgniter system files and vendor libraries.',
    rules: {
      preset: 'codeigniter',
      app_patterns: ['(^|/)application/'],
      framework_patterns: ['(^|/)system/'],
      external_patterns: ['(^|/)vendor/', '(^|/)(tests?|spec)/'],
    },
  },
  {
    id: 'fastapi',
    label: 'FastAPI',
    description: 'Prioritizes your app modules over FastAPI, Starlette, Pydantic and site-packages.',
    rules: {
      preset: 'fastapi',
      app_patterns: ['(^|/)(app|src|backend|service|api)/'],
      framework_patterns: ['(^|/)(fastapi|starlette|pydantic|uvicorn)/'],
      external_patterns: ['(^|/)(site-packages|dist-packages)/'],
    },
  },
  {
    id: 'django',
    label: 'Django',
    description: 'Highlights project apps over Django internals and site-packages.',
    rules: {
      preset: 'django',
      app_patterns: ['(^|/)(app|apps|project|src)/'],
      framework_patterns: ['(^|/)django/'],
      external_patterns: ['(^|/)(site-packages|dist-packages)/'],
    },
  },
  {
    id: 'laravel',
    label: 'Laravel',
    description: 'Highlights app code and de-emphasizes Laravel internals and vendor packages.',
    rules: {
      preset: 'laravel',
      app_patterns: ['(^|/)app/', '(^|/)(routes|database|config)/'],
      framework_patterns: ['(^|/)vendor/laravel/'],
      external_patterns: ['(^|/)vendor/'],
    },
  },
  {
    id: 'rails',
    label: 'Rails',
    description: 'Highlights Rails app files while keeping gems and Ruby internals secondary.',
    rules: {
      preset: 'rails',
      app_patterns: ['(^|/)(app|lib|config|db)/'],
      framework_patterns: ['(^|/)(actionpack|activerecord|activesupport)/'],
      external_patterns: ['(^|/)(gems|ruby/|vendor/bundle)/'],
    },
  },
  {
    id: 'express',
    label: 'Express / Node',
    description: 'Highlights source folders over node_modules dependencies.',
    rules: {
      preset: 'express',
      app_patterns: ['(^|/)(src|app|server|api|routes|controllers)/'],
      framework_patterns: ['(^|/)node_modules/(express|next|nestjs|koa)/'],
      external_patterns: ['(^|/)node_modules/'],
    },
  },
  {
    id: 'spring',
    label: 'Spring Boot',
    description: 'Highlights application packages over Spring and third-party jars.',
    rules: {
      preset: 'spring',
      app_patterns: ['(^|/)src/main/java/', '(^|/)src/main/kotlin/'],
      framework_patterns: ['(^|/)org/springframework/'],
      external_patterns: ['(^|/)(\\.m2/|gradle/caches/|jar:|BOOT-INF/lib/)'],
    },
  },
]

const DEFAULT_STACKTRACE_RULES = buildStacktraceRulesPreset('generic')

export function buildStacktraceRulesPreset(preset: string): StacktraceRules {
  const match = STACKTRACE_RULE_PRESETS.find(item => item.id === preset)
  if (!match) {
    return {
      preset: 'generic',
      app_patterns: [],
      framework_patterns: [],
      external_patterns: [],
    }
  }

  return {
    preset: match.rules.preset,
    app_patterns: [...match.rules.app_patterns],
    framework_patterns: [...match.rules.framework_patterns],
    external_patterns: [...match.rules.external_patterns],
  }
}

export function normalizeStacktraceRules(rules?: Partial<StacktraceRules> | null): StacktraceRules {
  const source = rules || DEFAULT_STACKTRACE_RULES
  return {
    preset: source.preset?.trim() || 'generic',
    app_patterns: normalizePatternList(source.app_patterns),
    framework_patterns: normalizePatternList(source.framework_patterns),
    external_patterns: normalizePatternList(source.external_patterns),
  }
}

function normalizePatternList(patterns?: string[] | null): string[] {
  return (patterns || []).map(pattern => pattern.trim()).filter(Boolean)
}

function matchesPattern(path: string, patterns: string[]): boolean {
  return patterns.some(pattern => {
    try {
      return new RegExp(pattern, 'i').test(path)
    } catch {
      return false
    }
  })
}

export function classifyStackFrame(filename: string, rules?: Partial<StacktraceRules> | null, inApp?: boolean): StackFrameKind {
  const normalizedPath = filename.replace(/\\/g, '/')
  const normalizedRules = normalizeStacktraceRules(rules)

  if (matchesPattern(normalizedPath, normalizedRules.app_patterns)) return 'app'
  if (matchesPattern(normalizedPath, normalizedRules.framework_patterns)) return 'framework'
  if (matchesPattern(normalizedPath, normalizedRules.external_patterns)) return 'external'
  if (inApp) return 'app'

  const lower = normalizedPath.toLowerCase()
  if (/(^|\/)(vendor|node_modules|site-packages|dist-packages)\//.test(lower)) return 'external'
  if (/(^|\/)(system|framework|gems|ruby|\.m2|gradle\/caches)\//.test(lower)) return 'framework'
  return 'external'
}
